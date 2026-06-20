// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

package pki_adcs_test

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/pro"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/testhelpers"
)

// NOTE (SDK BLOCKER): these acceptance tests will fail at the GET-after-write
// until the SDK expirationDate fix lands — AdcsCertificateResponse.ExpirationDate
// is *time.Time and cannot deserialize Jamf Pro's offset-less wire value
// ("2036-06-06T17:42:41"). See spike/SDK_PKI_EXPIRATION_DATE_FIX_PROMPT.md. The
// dummy self-signed certs themselves ARE accepted at create (validation is
// deferred to connection-test time, which the provider does not model).

const adcsAddr = "jamfplatform_pro_pki_adcs.test"

func testAccCheckAdcsDestroy(t *testing.T) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		c := pro.New(testhelpers.NewAcceptanceClient(t))
		ctx := context.Background()
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "jamfplatform_pro_pki_adcs" {
				continue
			}
			_, err := c.GetAdcsSettingsV1(ctx, rs.Primary.ID)
			if err != nil {
				if helpers.IsNotFoundError(err) {
					continue
				}
				return fmt.Errorf("error checking Jamf Pro AD CS integration %s: %s", rs.Primary.ID, err)
			}
			return fmt.Errorf("Jamf Pro AD CS integration %s still exists", rs.Primary.ID)
		}
		return nil
	}
}

// inboundConfig builds an INBOUND AD CS resource using the committed dummy certs.
func inboundConfig(name string, revocation bool, serverWoVersion, clientWoVersion int) string {
	return fmt.Sprintf(`
resource "jamfplatform_pro_pki_adcs" "test" {
  connector_mode     = "INBOUND"
  display_name       = %q
  ca_name            = "acc-ca"
  fqdn               = "adcs.example.com"
  adcs_url           = "connector.example.com"
  revocation_enabled = %t

  server_certificate = {
    data_wo    = filebase64("${path.module}/testdata/pki-dummy.pem")
    filename   = "server.pem"
    wo_version = %d
  }
  client_certificate = {
    data_wo     = filebase64("${path.module}/testdata/pki-dummy.p12")
    password_wo = "dummy-p12-password"
    filename    = "client.p12"
    wo_version  = %d
  }
}
`, name, revocation, serverWoVersion, clientWoVersion)
}

// outboundConfig builds an OUTBOUND AD CS resource fed by a same-apply api_role +
// api_client holding the AD CS certificate-job privileges.
func outboundConfig(name string) string {
	return fmt.Sprintf(`
resource "jamfplatform_pro_api_role" "adcs" {
  display_name = "%s-role"
  privileges   = ["Read AD CS Certificate Jobs", "Update AD CS Certificate Jobs"]
}

resource "jamfplatform_pro_api_client" "adcs" {
  display_name = "%s-client"
  api_roles    = [jamfplatform_pro_api_role.adcs.display_name]
  enabled      = true
}

resource "jamfplatform_pro_pki_adcs" "test" {
  connector_mode = "OUTBOUND"
  display_name   = %q
  ca_name        = "acc-ca"
  fqdn           = "adcs.example.com"
  api_client_id  = jamfplatform_pro_api_client.adcs.client_id
}
`, name, name, name)
}

// TestAccResource_ProPkiAdcs_Inbound drives create → merge update (rename + toggle
// revocation) → cert rotation (bump server wo_version) → import. Asserts both
// certificate metadata blocks are Computed-populated.
func TestAccResource_ProPkiAdcs_Inbound(t *testing.T) {
	testhelpers.AccPreCheck(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckAdcsDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: inboundConfig("tf-adcs-in", false, 1, 1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(adcsAddr, "id"),
					resource.TestCheckResourceAttr(adcsAddr, "connector_mode", "INBOUND"),
					resource.TestCheckResourceAttr(adcsAddr, "display_name", "tf-adcs-in"),
					resource.TestCheckResourceAttr(adcsAddr, "adcs_url", "connector.example.com"),
					resource.TestCheckResourceAttr(adcsAddr, "revocation_enabled", "false"),
					// Cert metadata blocks populated from GET-after-write.
					resource.TestCheckResourceAttrSet(adcsAddr, "server_certificate_details.serial_number"),
					resource.TestCheckResourceAttrSet(adcsAddr, "client_certificate_details.serial_number"),
					resource.TestCheckResourceAttr(adcsAddr, "client_certificate.filename", "client.p12"),
				),
			},
			{
				// Merge update: rename + flip revocation; certs unchanged (wo_version stable, preserved).
				Config: inboundConfig("tf-adcs-in-renamed", true, 1, 1),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(adcsAddr, plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(adcsAddr, "display_name", "tf-adcs-in-renamed"),
					resource.TestCheckResourceAttr(adcsAddr, "revocation_enabled", "true"),
				),
			},
			{
				// Rotate the server certificate by bumping its wo_version.
				Config: inboundConfig("tf-adcs-in-renamed", true, 2, 1),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(adcsAddr, plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(adcsAddr, "server_certificate.wo_version", "2"),
				),
			},
			{
				ResourceName:            adcsAddr,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"timeouts", "server_certificate", "client_certificate"},
			},
		},
	})
}

// TestAccResource_ProPkiAdcs_ModeFlipRequiresReplace verifies connector_mode is
// immutable: changing INBOUND → OUTBOUND forces resource replacement.
func TestAccResource_ProPkiAdcs_ModeFlipRequiresReplace(t *testing.T) {
	testhelpers.AccPreCheck(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckAdcsDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: inboundConfig("tf-adcs-flip", false, 1, 1),
				Check:  resource.TestCheckResourceAttr(adcsAddr, "connector_mode", "INBOUND"),
			},
			{
				Config: outboundConfig("tf-adcs-flip"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(adcsAddr, plancheck.ResourceActionDestroyBeforeCreate),
					},
				},
				Check: resource.TestCheckResourceAttr(adcsAddr, "connector_mode", "OUTBOUND"),
			},
		},
	})
}

// TestAccResource_ProPkiAdcs_Outbound creates an OUTBOUND integration fed by a
// same-apply api_role + api_client.
func TestAccResource_ProPkiAdcs_Outbound(t *testing.T) {
	testhelpers.AccPreCheck(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckAdcsDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: outboundConfig("tf-adcs-out"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(adcsAddr, "connector_mode", "OUTBOUND"),
					resource.TestCheckResourceAttrSet(adcsAddr, "api_client_id"),
				),
			},
		},
	})
}

// --- validator ExpectError tests (no tenant write — plan-time failures) ------
//
// Error-detail text wraps at ~80 cols; the messages use no-space tokens
// (connector_mode=) so the regex avoids a whitespace wrap point (see
// feedback_expecterror_regex_linewrap).

func TestAccResource_ProPkiAdcs_Inbound_ForbidsApiClientID(t *testing.T) {
	testhelpers.AccPreCheck(t)
	cfg := `
resource "jamfplatform_pro_pki_adcs" "test" {
  connector_mode = "INBOUND"
  display_name   = "tf-adcs-bad"
  ca_name        = "acc-ca"
  fqdn           = "adcs.example.com"
  adcs_url       = "connector.example.com"
  api_client_id  = "11111111-2222-3333-4444-555555555555"
  server_certificate = { data_wo = filebase64("${path.module}/testdata/pki-dummy.pem"), filename = "server.pem", wo_version = 1 }
  client_certificate = { data_wo = filebase64("${path.module}/testdata/pki-dummy.p12"), password_wo = "dummy-p12-password", filename = "client.p12", wo_version = 1 }
}
`
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: cfg, ExpectError: regexp.MustCompile(`api_client_id forbidden`)},
		},
	})
}

func TestAccResource_ProPkiAdcs_Inbound_RequiresServerCert(t *testing.T) {
	testhelpers.AccPreCheck(t)
	cfg := `
resource "jamfplatform_pro_pki_adcs" "test" {
  connector_mode = "INBOUND"
  display_name   = "tf-adcs-bad"
  ca_name        = "acc-ca"
  fqdn           = "adcs.example.com"
  adcs_url       = "connector.example.com"
  client_certificate = { data_wo = filebase64("${path.module}/testdata/pki-dummy.p12"), password_wo = "dummy-p12-password", filename = "client.p12", wo_version = 1 }
}
`
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: cfg, ExpectError: regexp.MustCompile(`server_certificate block required`)},
		},
	})
}

func TestAccResource_ProPkiAdcs_Outbound_RequiresApiClientID(t *testing.T) {
	testhelpers.AccPreCheck(t)
	cfg := `
resource "jamfplatform_pro_pki_adcs" "test" {
  connector_mode = "OUTBOUND"
  display_name   = "tf-adcs-out-bad"
  ca_name        = "acc-ca"
  fqdn           = "adcs.example.com"
}
`
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: cfg, ExpectError: regexp.MustCompile(`api_client_id required`)},
		},
	})
}

func TestAccResource_ProPkiAdcs_Outbound_ForbidsCertificates(t *testing.T) {
	testhelpers.AccPreCheck(t)
	cfg := `
resource "jamfplatform_pro_pki_adcs" "test" {
  connector_mode = "OUTBOUND"
  display_name   = "tf-adcs-bad"
  ca_name        = "acc-ca"
  fqdn           = "adcs.example.com"
  api_client_id  = "11111111-2222-3333-4444-555555555555"
  server_certificate = { data_wo = filebase64("${path.module}/testdata/pki-dummy.pem"), filename = "server.pem", wo_version = 1 }
}
`
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: cfg, ExpectError: regexp.MustCompile(`server_certificate block forbidden`)},
		},
	})
}
