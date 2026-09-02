// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

package sso_connection_test

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform"
	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/account"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/querycheck"
	"github.com/hashicorp/terraform-plugin-testing/querycheck/queryfilter"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/testhelpers"
)

// connectionResourceAddress is the address every test in this file uses for the
// subject resource.
const connectionResourceAddress = "jamfplatform_account_sso_connection.test"

// acceptanceDomainVariable names the environment variable holding a domain the
// acceptance organization has claimed *and verified*.
//
// It cannot be derived and it cannot be invented. A connection needs at least one
// verified domain, verification needs a public DNS record, and pointing a probe
// connection at one of the organization's live domains would repoint real
// sign-in. So the operator declares which domain is safe to attach a throwaway
// connection to, and these tests skip without it — the same declaration pattern
// the Security Cloud suites use for a scope only the operator can vouch for.
const acceptanceDomainVariable = "JAMFPLATFORM_ACC_SSO_VERIFIED_DOMAIN"

// probeConnectionID is an identifier no connection has. It is the target of the
// write-path probe below, which is deliberately aimed at something that does not
// exist.
const probeConnectionID = "con_tfaccprobe000001"

// upstreamErrorCode is the code Jamf answers every connection write with today.
// Restated because the Jamf Account SDK generates no error-code vocabulary; see
// mappings.go, which records the same exemption for the non-test code.
const upstreamErrorCode = "UPSTREAM_ERROR"

// acceptanceConnectionName builds a name unlikely to collide with a live
// connection, or with a leftover from an interrupted run.
func acceptanceConnectionName(part string) string {
	return "tf-acc-" + part + "-" + testhelpers.RunSuffix()
}

// verifiedDomain returns the declared verified domain, or skips.
func verifiedDomain(t *testing.T) string {
	t.Helper()
	domain := os.Getenv(acceptanceDomainVariable)
	if domain == "" {
		t.Skipf(
			"%s is not set. A Jamf Account SSO connection requires at least one domain the organization has "+
				"claimed and verified, verification requires a public DNS record this suite cannot publish, and "+
				"attaching a probe connection to one of the organization's live domains would repoint real "+
				"sign-in. Set it to a verified domain that is safe to attach a throwaway connection to.",
			acceptanceDomainVariable,
		)
	}
	return domain
}

// skipUnlessConnectionWritesWork is the gate every create-dependent test in this
// file opens with, and it exists because of a fault on Jamf's side rather than
// anything about this provider.
//
// Jamf refuses every connection create and every connection update with an
// internal failure carrying no detail, for every payload and in every region.
// Without this gate, these tests would fail rather than report — and a red suite
// that is red for a reason nobody can fix is worse than a skipped one, because it
// hides every genuine failure beside it.
//
// The probe is the single call that localises the fault: an update aimed at an
// identifier that does not exist. Reading or removing that identifier reports it
// missing, as it should; an update refuses it with the internal failure *before*
// Jamf looks the identifier up, which is a failure that cannot be about the
// request body. So the probe tells the two states apart with no verified domain,
// no throwaway connection, and nothing created even if it succeeds:
//
//   - the internal failure means the fault is still there, and the suite skips;
//   - a not-found means the write path has been fixed, and the suite runs;
//   - anything else is a third state nobody has seen, so the suite skips saying
//     what it saw rather than failing on an unexamined difference.
//
// Delete this function, and the calls to it, when Jamf fixes the write path. The
// tests below it are written to run unchanged when that happens.
func skipUnlessConnectionWritesWork(t *testing.T) {
	t.Helper()
	c := account.New(testhelpers.NewAcceptanceClient(t))

	_, err := c.UpdateConnection(context.Background(), probeConnectionID, probeConnectionRequest())
	if err == nil {
		t.Fatalf(
			"the write-path probe updated %s, an identifier no connection has. That should be impossible; "+
				"check whether a real connection is using this identifier before running the suite again.",
			probeConnectionID,
		)
	}

	apiErr := jamfplatform.AsAPIError(err)
	if apiErr == nil {
		t.Skipf("the write-path probe could not reach Jamf Account, so nothing here can be verified: %v", err)
	}
	for _, detail := range apiErr.Details() {
		if detail.Code == upstreamErrorCode {
			t.Skipf(
				"Jamf Account cannot create or change an SSO connection: the write path answers an internal "+
					"failure for every request, including this probe aimed at %s, an identifier no connection "+
					"has — while reading or removing that same identifier reports it missing as it should. The "+
					"fault is on Jamf's side and has been reported; these tests are correct and will run "+
					"unchanged once it clears. Trace identifier: %s. Reported by Jamf Account: %s",
				probeConnectionID, apiErr.TraceID, detail.Description,
			)
		}
	}
	if apiErr.HasStatus(404) {
		return
	}
	t.Skipf(
		"the write-path probe answered %d rather than the expected not-found or internal failure, which is a "+
			"state this suite has not seen. Skipping rather than failing on an unexamined difference. Body: %s",
		apiErr.StatusCode, apiErr.Body,
	)
}

// probeConnectionRequest is a complete, semantically valid generic OpenID
// Connect body for the write-path probe.
//
// It is maximal rather than minimal on purpose: a minimal body and a maximal one
// were both observed to be refused identically, so a full body rules out the
// reading that the probe merely omitted something Jamf needs.
func probeConnectionRequest() *account.ConnectionRequest {
	name := "tf-acc-write-probe"
	scopes := "openid email profile"
	return &account.ConnectionRequest{
		ConnectionType: account.ConnectionTypeOidc,
		Domains:        []string{"tf-acc-write-probe.example"},
		EnabledProducts: []account.EnabledProduct{{
			Product:        account.ProductAccount,
			EnabledTenants: []string{},
		}},
		Connection: account.ConnectionRequestConnection{
			OidcConnectionSettings: &account.OidcConnectionSettings{
				Name:                  name,
				Region:                account.RegionUs,
				ClientID:              stringPointer("tf-acc-write-probe-client"),
				ClientSecret:          stringPointer("tf-acc-write-probe-secret"),
				IssuerURL:             "idp.example",
				AuthorizationEndpoint: "idp.example/authorize",
				TokenEndpoint:         "idp.example/token",
				JwksUri:               "idp.example/keys",
				Scopes:                scopes,
				AliasLoginHintToIdp:   true,
			},
		},
	}
}

// stringPointer returns a pointer to a string, for the probe body's optional
// fields.
func stringPointer(s string) *string {
	return &s
}

// testAccCheckSSOConnectionDestroy verifies the connections made during the test
// were removed.
//
// It scans the organization's collection rather than reading each identifier,
// because Jamf is known to list a connection it cannot read on its own
// identifier — so a per-identifier check would report a connection as gone when
// Jamf still holds it.
func testAccCheckSSOConnectionDestroy(t *testing.T) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		c := account.New(testhelpers.NewAcceptanceClient(t))

		var wanted []string
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "jamfplatform_account_sso_connection" {
				continue
			}
			if id := rs.Primary.Attributes["id"]; id != "" {
				wanted = append(wanted, id)
			}
		}
		if len(wanted) == 0 {
			return nil
		}

		summaries, err := c.ListConnections(context.Background())
		if err != nil {
			return fmt.Errorf("error listing Jamf Account SSO connections: %w", err)
		}
		for _, summary := range summaries {
			for _, want := range wanted {
				if summary.ID == want {
					return fmt.Errorf("Jamf Account SSO connection %s still exists", want)
				}
			}
		}
		return nil
	}
}

// oidcConnectionConfig renders a generic OpenID Connect connection.
func oidcConnectionConfig(name, domain string) string {
	return fmt.Sprintf(`
		resource "jamfplatform_account_sso_connection" "test" {
			name            = %q
			connection_type = "generic_oidc"
			hosting_region  = "US"

			client_id     = "tf-acc-client"
			client_secret = "tf-acc-client-secret"
			scopes        = "openid email profile"

			domains = [%q]

			generic_oidc = {
				issuer_url             = "idp.example"
				authorization_endpoint = "idp.example/authorize"
				token_endpoint         = "idp.example/token"
				jwks_uri               = "idp.example/keys"
			}
		}
	`, name, domain)
}

// TestAccResource_AccountSSOConnection_Basic covers the create and the import
// round trip.
//
// Three of the assertions are worth stating the reasoning for.
//
// `display_name` is asserted set rather than equal to `name`, because whether
// Jamf appends a uniquifying suffix on this path is the one question no read
// could answer: eighteen of the twenty-two connections read carry one, and the
// write path is refused for every request. This is the highest-value thing to
// tighten the moment the fault clears — if the stored name equals the configured
// one, this becomes an equality check; if it does not, the two-attribute split is
// vindicated and the suffix pattern can be pinned here.
//
// `enabled_product_names` is asserted present rather than equal to anything,
// because it reports the products Jamf holds and the tenants of them are never
// returned. Asserting a value would be asserting half a fact.
//
// The import step ignores `enabled_products` and the write-only secret. Neither
// can be recovered by any read: the tenants are never echoed and the secret is
// never returned, which is exactly the drift blindness the resource documents.
func TestAccResource_AccountSSOConnection_Basic(t *testing.T) {
	testhelpers.AccPreCheckAccount(t)
	skipUnlessConnectionWritesWork(t)
	domain := verifiedDomain(t)
	name := acceptanceConnectionName("basic")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckSSOConnectionDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: oidcConnectionConfig(name, domain),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(connectionResourceAddress, "name", name),
					resource.TestCheckResourceAttr(connectionResourceAddress, "connection_type", "generic_oidc"),
					resource.TestCheckResourceAttr(connectionResourceAddress, "hosting_region", account.RegionUs),
					resource.TestCheckResourceAttr(connectionResourceAddress, "auth_method", "client_secret"),
					resource.TestCheckResourceAttr(connectionResourceAddress, "consent_flow", "false"),
					resource.TestCheckResourceAttr(connectionResourceAddress, "easy_config", "false"),
					resource.TestCheckResourceAttr(connectionResourceAddress, "domains.#", "1"),
					resource.TestCheckTypeSetElemAttr(connectionResourceAddress, "domains.*", domain),
					resource.TestCheckResourceAttr(connectionResourceAddress, "generic_oidc.issuer_url", "idp.example"),
					resource.TestCheckResourceAttrSet(connectionResourceAddress, "id"),
					resource.TestCheckResourceAttrSet(connectionResourceAddress, "display_name"),
					resource.TestCheckResourceAttrSet(connectionResourceAddress, "enabled_product_names.#"),
					resource.TestCheckNoResourceAttr(connectionResourceAddress, "client_secret"),
				),
			},
			{
				ResourceName:      connectionResourceAddress,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"timeouts",
					"client_secret",
					"client_secret_wo_version",
					"enabled_products",
					"enabled_environments",
				},
			},
		},
	})
}

// TestAccResource_AccountSSOConnection_Update covers the change path, and with it
// the two behaviours the specification describes and no probe could confirm: that
// an update replaces the settings rather than patching them, and that omitting the
// client secret keeps the stored one.
//
// The second step changes the name and adds an optional value while leaving the
// secret out. If an update really replaces, the value left out of the first step
// has to survive; if it really keeps the secret, the apply has to succeed at all,
// since a connection whose secret was cleared could not be written back.
func TestAccResource_AccountSSOConnection_Update(t *testing.T) {
	testhelpers.AccPreCheckAccount(t)
	skipUnlessConnectionWritesWork(t)
	domain := verifiedDomain(t)
	name := acceptanceConnectionName("update")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckSSOConnectionDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: oidcConnectionConfig(name, domain),
			},
			{
				Config: fmt.Sprintf(`
					resource "jamfplatform_account_sso_connection" "test" {
						name            = %q
						connection_type = "generic_oidc"
						hosting_region  = "US"

						client_id = "tf-acc-client"
						scopes    = "openid email profile groups"

						send_nonce               = true
						sync_attributes_at_login = false
						omit_login_hint          = true
						pkce                     = "s256"

						session_duration_minutes   = 480
						inactivity_timeout_minutes = 30

						group_name_filter = {
							operator = "and"
							groups   = ["tf-acc-one", "tf-acc-two"]
						}

						domains = [%q]

						generic_oidc = {
							issuer_url             = "idp.example"
							authorization_endpoint = "idp.example/authorize"
							token_endpoint         = "idp.example/token"
							jwks_uri               = "idp.example/keys"
							user_info_endpoint     = "idp.example/userinfo"
						}
					}
				`, name+"-changed", domain),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(connectionResourceAddress, plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(connectionResourceAddress, "name", name+"-changed"),
					resource.TestCheckResourceAttr(connectionResourceAddress, "send_nonce", "true"),
					resource.TestCheckResourceAttr(connectionResourceAddress, "sync_attributes_at_login", "false"),
					resource.TestCheckResourceAttr(connectionResourceAddress, "omit_login_hint", "true"),
					resource.TestCheckResourceAttr(connectionResourceAddress, "pkce", "s256"),
					resource.TestCheckResourceAttr(connectionResourceAddress, "session_duration_minutes", "480"),
					resource.TestCheckResourceAttr(connectionResourceAddress, "inactivity_timeout_minutes", "30"),
					resource.TestCheckResourceAttr(connectionResourceAddress, "group_name_filter.operator", "and"),
					resource.TestCheckResourceAttr(connectionResourceAddress, "group_name_filter.groups.#", "2"),
					resource.TestCheckResourceAttr(connectionResourceAddress, "generic_oidc.user_info_endpoint", "idp.example/userinfo"),
				),
			},
		},
	})
}

// TestAccResource_AccountSSOConnection_EmptyGroupFilterRoundTrips pins the
// distinction the group filter block exists to preserve, on the only surface that
// can actually prove it: an operator with no groups is a real value and is
// different from the filter being absent. Every connection read carried the
// former, so getting this wrong would rewrite live configuration.
func TestAccResource_AccountSSOConnection_EmptyGroupFilterRoundTrips(t *testing.T) {
	testhelpers.AccPreCheckAccount(t)
	skipUnlessConnectionWritesWork(t)
	domain := verifiedDomain(t)
	name := acceptanceConnectionName("filter")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckSSOConnectionDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "jamfplatform_account_sso_connection" "test" {
						name            = %q
						connection_type = "generic_oidc"
						hosting_region  = "US"

						client_id     = "tf-acc-client"
						client_secret = "tf-acc-client-secret"
						scopes        = "openid email profile"

						group_name_filter = {
							operator = "or"
							groups   = []
						}

						domains = [%q]

						generic_oidc = {
							issuer_url             = "idp.example"
							authorization_endpoint = "idp.example/authorize"
							token_endpoint         = "idp.example/token"
							jwks_uri               = "idp.example/keys"
						}
					}
				`, name, domain),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(connectionResourceAddress, "group_name_filter.operator", "or"),
					resource.TestCheckResourceAttr(connectionResourceAddress, "group_name_filter.groups.#", "0"),
				),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
	})
}

// TestAccResource_AccountSSOConnection_ImmutableAttributesReplace stands in for
// the two changes an update cannot make. Both are specification-derived: Jamf says
// an update can move a connection to neither another family nor another region,
// and the refusal was never observed because every write is refused.
//
// So this asserts what the provider plans rather than what Jamf answers, which is
// the honest thing to assert either way — a replacement is correct whether Jamf
// would have refused the in-place change or silently ignored it.
func TestAccResource_AccountSSOConnection_ImmutableAttributesReplace(t *testing.T) {
	testhelpers.AccPreCheckAccount(t)
	skipUnlessConnectionWritesWork(t)
	domain := verifiedDomain(t)
	name := acceptanceConnectionName("replace")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckSSOConnectionDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: oidcConnectionConfig(name, domain),
			},
			{
				Config: fmt.Sprintf(`
					resource "jamfplatform_account_sso_connection" "test" {
						name            = %q
						connection_type = "okta"
						hosting_region  = "US"

						client_id     = "tf-acc-client"
						client_secret = "tf-acc-client-secret"
						scopes        = "openid email profile"

						domains = [%q]

						okta = {
							domain = "tf-acc.okta.example"
						}
					}
				`, name, domain),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(connectionResourceAddress, plancheck.ResourceActionDestroyBeforeCreate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(connectionResourceAddress, "connection_type", "okta"),
					resource.TestCheckResourceAttrSet(connectionResourceAddress, "okta.issuer_url"),
					resource.TestCheckResourceAttrSet(connectionResourceAddress, "okta.token_endpoint"),
				),
			},
		},
	})
}

// TestAccResource_AccountSSOConnection_Disappears covers the drift path: a
// connection removed out from under Terraform has to be dropped from state and
// planned afresh rather than failing the refresh.
//
// It is worth its own test here because the not-found on a single connection is
// deliberately *not* trusted on its own — Jamf is known to list a connection it
// cannot read on its identifier, and the provider keeps such a connection in
// state. So this exercises the other branch of that decision, where the
// collection agrees the connection is gone.
func TestAccResource_AccountSSOConnection_Disappears(t *testing.T) {
	testhelpers.AccPreCheckAccount(t)
	skipUnlessConnectionWritesWork(t)
	domain := verifiedDomain(t)
	name := acceptanceConnectionName("gone")

	var connectionID string

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckSSOConnectionDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: oidcConnectionConfig(name, domain),
				Check: resource.ComposeAggregateTestCheckFunc(
					captureAttribute(connectionResourceAddress, "id", &connectionID),
				),
			},
			{
				PreConfig: func() {
					c := account.New(testhelpers.NewAcceptanceClient(t))
					if err := c.DeleteConnection(context.Background(), connectionID); err != nil {
						t.Fatalf("drift preconfig: removing SSO connection %s: %v", connectionID, err)
					}
				},
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

// TestAccResource_AccountSSOConnection_Query covers the list resource against a
// connection this test made, which is the only way to assert on a known entry:
// the organization's own connections are live configuration and their names are
// not this suite's to depend on.
func TestAccResource_AccountSSOConnection_Query(t *testing.T) {
	testhelpers.AccPreCheckAccount(t)
	skipUnlessConnectionWritesWork(t)
	domain := verifiedDomain(t)
	name := acceptanceConnectionName("query")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckSSOConnectionDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: oidcConnectionConfig(name, domain),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(connectionResourceAddress, "id"),
				),
			},
			{
				Query: true,
				Config: `
					provider "jamfplatform" {}

					list "jamfplatform_account_sso_connection" "test" {
						provider         = jamfplatform
						include_resource = true

						config {}
					}
				`,
				QueryResultChecks: []querycheck.QueryResultCheck{
					querycheck.ExpectResourceKnownValues(
						"jamfplatform_account_sso_connection.test",
						queryfilter.ByDisplayName(knownvalue.StringExact(name)),
						[]querycheck.KnownValueCheck{
							{Path: tfjsonpath.New("connection_type"), KnownValue: knownvalue.StringExact("generic_oidc")},
							{Path: tfjsonpath.New("hosting_region"), KnownValue: knownvalue.StringExact(account.RegionUs)},
						},
					),
				},
			},
		},
	})
}

// --- plan-time refusals, which need no write and so run whatever Jamf is doing ---

// TestAccResource_AccountSSOConnection_MismatchedSettingsBlockRefusedAtPlan is
// the most valuable of the plan-time tests, because it is the mistake Jamf
// handles worst: a settings block disagreeing with the declared family is
// accepted and then fails with an internal failure naming nothing, which is
// indistinguishable from any other failure.
func TestAccResource_AccountSSOConnection_MismatchedSettingsBlockRefusedAtPlan(t *testing.T) {
	testhelpers.AccPreCheckAccount(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
					resource "jamfplatform_account_sso_connection" "test" {
						name            = "tf-acc-mismatch"
						connection_type = "generic_oidc"
						hosting_region  = "US"
						client_id       = "tf-acc-client"
						scopes          = "openid"
						domains         = ["tf-acc.example"]

						entra = {
							domain        = "contoso.example"
							tenant_domain = "contoso.example"
						}
					}
				`,
				ExpectError: regexpBlockDoesNotMatch,
			},
		},
	})
}

// TestAccResource_AccountSSOConnection_MissingSettingsBlockRefusedAtPlan is the
// other half of the same rule.
func TestAccResource_AccountSSOConnection_MissingSettingsBlockRefusedAtPlan(t *testing.T) {
	testhelpers.AccPreCheckAccount(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
					resource "jamfplatform_account_sso_connection" "test" {
						name            = "tf-acc-missing-block"
						connection_type = "okta"
						hosting_region  = "US"
						client_id       = "tf-acc-client"
						scopes          = "openid"
						domains         = ["tf-acc.example"]
					}
				`,
				ExpectError: regexpBlockRequired,
			},
		},
	})
}

// TestAccResource_AccountSSOConnection_ScopesRefusedForEntraAtPlan drives the
// rule that came from the survey rather than from documentation: no Entra
// connection read carried any scopes, and the settings Jamf accepts for one have
// nowhere to put them.
func TestAccResource_AccountSSOConnection_ScopesRefusedForEntraAtPlan(t *testing.T) {
	testhelpers.AccPreCheckAccount(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
					resource "jamfplatform_account_sso_connection" "test" {
						name            = "tf-acc-entra-scopes"
						connection_type = "entra"
						hosting_region  = "US"
						client_id       = "tf-acc-client"
						scopes          = "openid"
						domains         = ["tf-acc.example"]

						entra = {
							domain        = "contoso.example"
							tenant_domain = "contoso.example"
						}
					}
				`,
				ExpectError: regexpScopesRefused,
			},
		},
	})
}

// TestAccResource_AccountSSOConnection_SecretRefusedWithSignedAssertionAtPlan
// drives the one rule Jamf's own documentation is explicit about.
func TestAccResource_AccountSSOConnection_SecretRefusedWithSignedAssertionAtPlan(t *testing.T) {
	testhelpers.AccPreCheckAccount(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
					resource "jamfplatform_account_sso_connection" "test" {
						name            = "tf-acc-jwt-secret"
						connection_type = "generic_oidc"
						hosting_region  = "US"
						auth_method     = "private_key_jwt"
						client_id       = "tf-acc-client"
						client_secret   = "tf-acc-client-secret"
						scopes          = "openid"
						domains         = ["tf-acc.example"]

						generic_oidc = {
							issuer_url             = "idp.example"
							authorization_endpoint = "idp.example/authorize"
							token_endpoint         = "idp.example/token"
							jwks_uri               = "idp.example/keys"
						}
					}
				`,
				ExpectError: regexpSecretRefused,
			},
		},
	})
}

// TestAccResource_AccountSSOConnection_MixedCaseDomainRefusedAtPlan pins the
// normalisation guard on the domain names. Jamf holds a claimed domain in lower
// case, so a mixed-case entry names a domain the organization does not hold under
// that spelling.
func TestAccResource_AccountSSOConnection_MixedCaseDomainRefusedAtPlan(t *testing.T) {
	testhelpers.AccPreCheckAccount(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
					resource "jamfplatform_account_sso_connection" "test" {
						name            = "tf-acc-mixed-case"
						connection_type = "generic_oidc"
						hosting_region  = "US"
						client_id       = "tf-acc-client"
						scopes          = "openid"
						domains         = ["TF-Acc.Example"]

						generic_oidc = {
							issuer_url             = "idp.example"
							authorization_endpoint = "idp.example/authorize"
							token_endpoint         = "idp.example/token"
							jwks_uri               = "idp.example/keys"
						}
					}
				`,
				ExpectError: regexpMixedCaseDomain,
			},
		},
	})
}

// TestAccResource_AccountSSOConnection_EmptyDomainsRefusedAtPlan pins the one
// collection constraint Jamf does report by name, caught at plan time so the
// operator does not have to read a refusal to learn it.
func TestAccResource_AccountSSOConnection_EmptyDomainsRefusedAtPlan(t *testing.T) {
	testhelpers.AccPreCheckAccount(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
					resource "jamfplatform_account_sso_connection" "test" {
						name            = "tf-acc-no-domains"
						connection_type = "generic_oidc"
						hosting_region  = "US"
						client_id       = "tf-acc-client"
						scopes          = "openid"
						domains         = []

						generic_oidc = {
							issuer_url             = "idp.example"
							authorization_endpoint = "idp.example/authorize"
							token_endpoint         = "idp.example/token"
							jwks_uri               = "idp.example/keys"
						}
					}
				`,
				ExpectError: regexpAtLeastOneDomain,
			},
		},
	})
}

// TestAccResource_AccountSSOConnection_UnknownRegionRefusedAtPlan pins the
// vocabulary check. Jamf names the offending value in its own refusal but never
// the field it was on, so catching it at plan time is what turns an unattributed
// failure into something a practitioner can act on.
func TestAccResource_AccountSSOConnection_UnknownRegionRefusedAtPlan(t *testing.T) {
	testhelpers.AccPreCheckAccount(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
					resource "jamfplatform_account_sso_connection" "test" {
						name            = "tf-acc-bad-region"
						connection_type = "generic_oidc"
						hosting_region  = "MARS"
						client_id       = "tf-acc-client"
						scopes          = "openid"
						domains         = ["tf-acc.example"]

						generic_oidc = {
							issuer_url             = "idp.example"
							authorization_endpoint = "idp.example/authorize"
							token_endpoint         = "idp.example/token"
							jwks_uri               = "idp.example/keys"
						}
					}
				`,
				ExpectError: regexpInvalidValue,
			},
		},
	})
}

// captureAttribute records an applied attribute so a later step can act on it.
// PreConfig runs without access to state, so a value has to be carried out of the
// apply step this way.
func captureAttribute(address, name string, into *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[address]
		if !ok {
			return fmt.Errorf("resource %s not found in state", address)
		}
		value, ok := rs.Primary.Attributes[name]
		if !ok {
			return fmt.Errorf("resource %s has no %s attribute", address, name)
		}
		*into = value
		return nil
	}
}

// Expected-error patterns for the plan-time refusals. Terraform wraps diagnostic
// text at roughly 80 columns, so each pattern matches a short phrase that cannot
// be split across a line break.
var (
	regexpBlockDoesNotMatch = regexp.MustCompile(`does not match this connection type`)
	regexpBlockRequired     = regexp.MustCompile(`required for this connection type`)
	regexpScopesRefused     = regexp.MustCompile(`not accepted for an Entra`)
	regexpSecretRefused     = regexp.MustCompile(`no shared secret`)
	regexpMixedCaseDomain   = regexp.MustCompile(`must be lower case`)
	regexpAtLeastOneDomain  = regexp.MustCompile(`at least 1 element`)
	regexpInvalidValue      = regexp.MustCompile(`Invalid Attribute Value Match`)
)
