// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

package user_initiated_enrollment_settings_test

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/pro"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/testhelpers"
)

// The User-Initiated Enrollment settings object is a tenant-wide singleton that
// always exists and cannot be deleted (Delete is state-only by design). Every
// test uses an INVERTED CheckDestroy: after `terraform destroy` the record must
// still be readable on the tenant via GetEnrollmentSettingsV4.
//
// The /v4 PUT is a symmetric full-replace over its scalar fields, so the
// resource read-merges and round-trips fields it does not own (notably the six
// Re-enrollment flush fields). The coexistence test below is the load-bearing
// proof that the /v4 write does not clobber the /v1 Re-enrollment record and
// vice versa.
//
// Env-gated tests:
//   - TestAccResource_..._AccessGroupLdap requires a real configured LDAP /
//     directory-service group on the tenant, supplied via:
//       JAMFPLATFORM_ACC_ENROLLMENT_LDAP_SERVER_ID
//       JAMFPLATFORM_ACC_ENROLLMENT_GROUP_ID
//       JAMFPLATFORM_ACC_ENROLLMENT_GROUP_NAME
//     Unset → t.Skipf (mirrors sso_settings' requireIdpURL pattern).

const uieResourceAddr = "jamfplatform_pro_user_initiated_enrollment_settings.test"

const (
	envEnrollmentLdapServerID = "JAMFPLATFORM_ACC_ENROLLMENT_LDAP_SERVER_ID"
	envEnrollmentGroupName    = "JAMFPLATFORM_ACC_ENROLLMENT_GROUP_NAME"
)

// fixtureKeystoreBase64 reads the committed dummy PKCS#12 fixture and returns
// its raw base64. The keystore field is WriteOnly so it cannot be a file path in
// HCL (acc configs run from a Terraform temp dir); we inject the base64 string
// directly into the config. Go test code runs with cwd = package dir, so the
// relative testdata path resolves without runtime.Caller gymnastics.
func fixtureKeystoreBase64(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile("testdata/signing.p12")
	if err != nil {
		t.Fatalf("reading keystore fixture: %v", err)
	}
	return base64.StdEncoding.EncodeToString(data)
}

// requireLdapGroupEnv skips the test unless all three LDAP-group env vars are
// set, returning (ldapServerID, groupID, groupName).
// checkAccessGroupResolvedID finds the access_group element with the given name
// and asserts the provider resolved a directory group id for it — non-empty and
// not the built-in "-1". The value itself is tenant-specific and not asserted.
func checkAccessGroupResolvedID(addr, groupName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[addr]
		if !ok {
			return fmt.Errorf("resource %s not found in state", addr)
		}
		nameKey := regexp.MustCompile(`^access_group\.(\d+)\.name$`)
		for k, v := range rs.Primary.Attributes {
			m := nameKey.FindStringSubmatch(k)
			if m == nil || v != groupName {
				continue
			}
			gid := rs.Primary.Attributes[fmt.Sprintf("access_group.%s.directory_service_group_id", m[1])]
			if gid == "" || gid == "-1" {
				return fmt.Errorf("access_group %q: resolved directory_service_group_id = %q, want a non-empty resolved id", groupName, gid)
			}
			return nil
		}
		return fmt.Errorf("access_group %q not found in state for %s", groupName, addr)
	}
}

func requireLdapGroupEnv(t *testing.T) (string, string) {
	t.Helper()
	srv := os.Getenv(envEnrollmentLdapServerID)
	name := os.Getenv(envEnrollmentGroupName)
	if srv == "" || name == "" {
		t.Skipf("skipping: set %s and %s to a real directory-service group (by name) to exercise the LDAP access-group add/remove path",
			envEnrollmentLdapServerID, envEnrollmentGroupName)
	}
	return srv, name
}

// checkUIEStillExists verifies Delete did not remove the settings record.
func checkUIEStillExists(t *testing.T) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		c := pro.New(testhelpers.NewAcceptanceClient(t))
		got, err := c.GetEnrollmentSettingsV4(context.Background())
		if err != nil {
			return fmt.Errorf("expected User-Initiated Enrollment settings record to persist after destroy: %w", err)
		}
		if got == nil {
			return fmt.Errorf("expected non-nil User-Initiated Enrollment settings post-destroy")
		}
		return nil
	}
}

// checkReEnrollmentStillExistsUIE is the inverted destroy check for the
// coexistence test's Re-enrollment resource.
func checkReEnrollmentStillExistsUIE(t *testing.T) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		c := pro.New(testhelpers.NewAcceptanceClient(t))
		got, err := c.GetReenrollmentSettingsV1(context.Background())
		if err != nil {
			return fmt.Errorf("expected Re-enrollment settings record to persist after destroy: %w", err)
		}
		if got == nil {
			return fmt.Errorf("expected non-nil Re-enrollment settings post-destroy")
		}
		return nil
	}
}

// uieScalarConfig renders a scalar-only config. signing_mdm_profile_enabled is
// deliberately NOT a parameter: the mdmSigningCertificateRequiredValidator fails
// the plan whenever it is true with no mdm_signing_certificate block, so the
// scalar-toggle tests keep it false (its true-path is covered by the MDM cert
// test). No certs and no access_group block are declared here, keeping the
// config import-safe and deterministic.
func uieScalarConfig(b bool, managementUsername string) string {
	return fmt.Sprintf(`
		resource "jamfplatform_pro_user_initiated_enrollment_settings" "test" {
			skip_certificate_installation = %[1]t
			restrict_reenrollment         = %[1]t

			enable_computer_enrollment             = %[1]t
			create_management_account              = %[1]t
			management_username                    = %[2]q
			hide_management_account                = %[1]t
			allow_ssh_only_management_account      = %[1]t
			ensure_ssh_running                     = %[1]t
			launch_self_service                    = %[1]t
			sign_quickadd_package                  = %[1]t
			account_driven_device_enrollment_macos = %[1]t

			profile_driven_enrollment_via_url_institutional = %[1]t
			profile_driven_enrollment_via_url_personal      = %[1]t
			account_driven_user_enrollment                  = %[1]t
			account_driven_user_enrollment_visionos         = %[1]t
			merge_managed_apple_account_usernames           = %[1]t
			account_driven_device_enrollment_ios            = %[1]t
			account_driven_device_enrollment_visionos       = %[1]t
		}
	`, b, managementUsername)
}

// uieScalarChecks asserts every scalar toggle equals b and management_username
// equals the supplied value.
func uieScalarChecks(b bool, managementUsername string) resource.TestCheckFunc {
	v := fmt.Sprintf("%t", b)
	return resource.ComposeAggregateTestCheckFunc(
		resource.TestCheckResourceAttr(uieResourceAddr, "id", "singleton"),
		resource.TestCheckResourceAttr(uieResourceAddr, "skip_certificate_installation", v),
		resource.TestCheckResourceAttr(uieResourceAddr, "restrict_reenrollment", v),
		resource.TestCheckResourceAttr(uieResourceAddr, "enable_computer_enrollment", v),
		resource.TestCheckResourceAttr(uieResourceAddr, "create_management_account", v),
		resource.TestCheckResourceAttr(uieResourceAddr, "management_username", managementUsername),
		resource.TestCheckResourceAttr(uieResourceAddr, "hide_management_account", v),
		resource.TestCheckResourceAttr(uieResourceAddr, "allow_ssh_only_management_account", v),
		resource.TestCheckResourceAttr(uieResourceAddr, "ensure_ssh_running", v),
		resource.TestCheckResourceAttr(uieResourceAddr, "launch_self_service", v),
		resource.TestCheckResourceAttr(uieResourceAddr, "sign_quickadd_package", v),
		resource.TestCheckResourceAttr(uieResourceAddr, "account_driven_device_enrollment_macos", v),
		resource.TestCheckResourceAttr(uieResourceAddr, "profile_driven_enrollment_via_url_institutional", v),
		resource.TestCheckResourceAttr(uieResourceAddr, "profile_driven_enrollment_via_url_personal", v),
		resource.TestCheckResourceAttr(uieResourceAddr, "account_driven_user_enrollment", v),
		resource.TestCheckResourceAttr(uieResourceAddr, "account_driven_user_enrollment_visionos", v),
		resource.TestCheckResourceAttr(uieResourceAddr, "merge_managed_apple_account_usernames", v),
		resource.TestCheckResourceAttr(uieResourceAddr, "account_driven_device_enrollment_ios", v),
		resource.TestCheckResourceAttr(uieResourceAddr, "account_driven_device_enrollment_visionos", v),
	)
}

// TestAccResource_ProUserInitiatedEnrollmentSettings_Update drives the scalar
// Update round-trip. Step 1 sets a baseline of every General/Computers/Devices
// scalar toggle to false + a management username; step 2 flips every scalar
// toggle to true and changes the username. signing_mdm_profile_enabled is
// excluded from the flip set (its true-value requires a cert block — see the
// MDM cert test).
func TestAccResource_ProUserInitiatedEnrollmentSettings_Update(t *testing.T) {
	testhelpers.AccPreCheck(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             checkUIEStillExists(t),
		Steps: []resource.TestStep{
			{
				Config: uieScalarConfig(false, "tf-acc-lapsadmin"),
				Check:  uieScalarChecks(false, "tf-acc-lapsadmin"),
			},
			{
				Config: uieScalarConfig(true, "tf-acc-lapsadmin2"),
				Check:  uieScalarChecks(true, "tf-acc-lapsadmin2"),
			},
		},
	})
}

// TestAccResource_ProUserInitiatedEnrollmentSettings_Import exercises the import
// round-trip from a minimal scalar config with no sub-blocks. ImportStateVerify
// is safe only because the config declares no Optional sub-blocks (access_group,
// certs): the singleton importer's post-import Read populates the base scalars,
// which match. The second step asserts the non-singleton import guard.
func TestAccResource_ProUserInitiatedEnrollmentSettings_Import(t *testing.T) {
	testhelpers.AccPreCheck(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             checkUIEStillExists(t),
		Steps: []resource.TestStep{
			{
				Config: uieScalarConfig(false, "tf-acc-import"),
			},
			{
				ResourceName:      uieResourceAddr,
				ImportState:       true,
				ImportStateId:     "singleton",
				ImportStateVerify: true,
			},
			{
				ResourceName:  uieResourceAddr,
				ImportState:   true,
				ImportStateId: "not-the-singleton",
				ExpectError:   regexp.MustCompile(`Invalid singleton import identifier`),
			},
		},
	})
}

// TestAccResource_ProUserInitiatedEnrollmentSettings_SigningCertRequiredValidator
// verifies the plan-time mdmSigningCertificateRequiredValidator: enabling the
// third-party signing toggle with no mdm_signing_certificate block must fail at
// plan time. The regex matches the contiguous attribute name token in the
// diagnostic summary ("mdm_signing_certificate required when…"), avoiding any
// space that Terraform's ~80-col wrapping might break.
func TestAccResource_ProUserInitiatedEnrollmentSettings_SigningCertRequiredValidator(t *testing.T) {
	testhelpers.AccPreCheck(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
					resource "jamfplatform_pro_user_initiated_enrollment_settings" "test" {
						signing_mdm_profile_enabled = true
					}
				`,
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`mdm_signing_certificate`),
			},
		},
	})
}

// TestAccResource_ProUserInitiatedEnrollmentSettings_MdmSigningCert uploads the
// dummy keystore with signing_mdm_profile_enabled = true (uploading the MDM cert
// requires the toggle true in the same apply, else 400). The dummy self-signed
// cert populated subject in wire probes, so we assert mdm_signing_certificate.
// subject is non-empty. WriteOnly inputs (keystore_file/keystore_password) and
// keystore_password_wo_version never land in state and are not asserted. Step 2
// disables the toggle (removing the cert) and asserts it applies clean.
func TestAccResource_ProUserInitiatedEnrollmentSettings_MdmSigningCert(t *testing.T) {
	testhelpers.AccPreCheck(t)
	keystore := fixtureKeystoreBase64(t)

	withCert := fmt.Sprintf(`
		resource "jamfplatform_pro_user_initiated_enrollment_settings" "test" {
			signing_mdm_profile_enabled = true
			mdm_signing_certificate = {
				keystore_file                = %q
				keystore_password            = "TestPass123"
				keystore_password_wo_version = 1
			}
		}
	`, keystore)

	const withoutCert = `
		resource "jamfplatform_pro_user_initiated_enrollment_settings" "test" {
			signing_mdm_profile_enabled = false
		}
	`

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             checkUIEStillExists(t),
		Steps: []resource.TestStep{
			{
				Config: withCert,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(uieResourceAddr, "signing_mdm_profile_enabled", "true"),
					resource.TestCheckResourceAttrWith(uieResourceAddr, "mdm_signing_certificate.subject", func(v string) error {
						if v == "" {
							return fmt.Errorf("expected mdm_signing_certificate.subject to be populated after upload, got empty")
						}
						return nil
					}),
				),
			},
			{
				Config: withoutCert,
				Check:  resource.TestCheckResourceAttr(uieResourceAddr, "signing_mdm_profile_enabled", "false"),
			},
		},
	})
}

// TestAccResource_ProUserInitiatedEnrollmentSettings_DeveloperCert verifies
// sign_quickadd_package toggles on WITHOUT a developer certificate — it has no
// cert-required invariant (unlike the MDM signing cert), so the apply succeeds.
//
// The developer_certificate UPLOAD path is deliberately NOT acc-tested with the
// committed dummy keystore. The shared upload code (buildCertificateIdentity /
// applyCertToBody) IS exercised end-to-end by the MDM signing certificate test.
// The dev slot specifically expects a real Apple Developer ID Application
// certificate: a self-signed .p12 is accepted by the PUT but not ingested
// (developer_certificate.subject / serial_number stay empty), which leaves those
// Computed attributes resolving unknown and would produce a perpetual diff. A
// clean dev-cert upload acc test requires a genuine Apple Developer ID .p12
// supplied out of band.
func TestAccResource_ProUserInitiatedEnrollmentSettings_DeveloperCert(t *testing.T) {
	testhelpers.AccPreCheck(t)

	config := `
		resource "jamfplatform_pro_user_initiated_enrollment_settings" "test" {
			sign_quickadd_package = true
		}
	`

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             checkUIEStillExists(t),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check:  resource.TestCheckResourceAttr(uieResourceAddr, "sign_quickadd_package", "true"),
			},
		},
	})
}

// TestAccResource_ProUserInitiatedEnrollmentSettings_AccessGroupBuiltin manages
// the built-in "All Directory Service Users" group (directory_service_group_id
// = "-1", ldap_server_id = "-1"), which exists on every tenant and cannot be
// created or deleted — only its toggles are editable. Step 2 flips the three
// toggles. require_eula is intentionally never set: the server overrides it,
// producing a perpetual diff (documented limitation). Asserted via
// TestCheckTypeSetElemNestedAttrs.
//
// NOTE: the access-group reconcile replaces to the exact declared set (it
// deletes current groups the plan no longer references, except the built-in).
// These tests assume the tenant has no pre-existing non-built-in access groups;
// a tenant with extra groups would see them deleted by this config.
func TestAccResource_ProUserInitiatedEnrollmentSettings_AccessGroupBuiltin(t *testing.T) {
	testhelpers.AccPreCheck(t)

	builtinConfig := func(enterprise, personal, adue bool) string {
		return fmt.Sprintf(`
			resource "jamfplatform_pro_user_initiated_enrollment_settings" "test" {
				access_group = [{
					ldap_server_id                         = "-1"
					name                                   = "All Directory Service Users"
					enterprise_enrollment_enabled          = %t
					personal_enrollment_enabled            = %t
					account_driven_user_enrollment_enabled = %t
				}]
			}
		`, enterprise, personal, adue)
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             checkUIEStillExists(t),
		Steps: []resource.TestStep{
			{
				Config: builtinConfig(true, false, true),
				Check: resource.TestCheckTypeSetElemNestedAttrs(uieResourceAddr, "access_group.*", map[string]string{
					"directory_service_group_id":             "-1",
					"ldap_server_id":                         "-1",
					"name":                                   "All Directory Service Users",
					"enterprise_enrollment_enabled":          "true",
					"personal_enrollment_enabled":            "false",
					"account_driven_user_enrollment_enabled": "true",
				}),
			},
			{
				Config: builtinConfig(false, true, false),
				Check: resource.TestCheckTypeSetElemNestedAttrs(uieResourceAddr, "access_group.*", map[string]string{
					"directory_service_group_id":             "-1",
					"ldap_server_id":                         "-1",
					"enterprise_enrollment_enabled":          "false",
					"personal_enrollment_enabled":            "true",
					"account_driven_user_enrollment_enabled": "false",
				}),
			},
		},
	})
}

// TestAccResource_ProUserInitiatedEnrollmentSettings_AccessGroupLdap exercises
// the nested-collection add AND remove path against a real configured
// directory-service group. Env-gated (skips when the three env vars are unset).
// Step 1 declares the built-in group plus the env group (ADD); step 2 declares
// only the built-in (REMOVE the env group). Set membership is asserted on each
// step.
func TestAccResource_ProUserInitiatedEnrollmentSettings_AccessGroupLdap(t *testing.T) {
	testhelpers.AccPreCheck(t)
	ldapServerID, groupName := requireLdapGroupEnv(t)

	const builtinElem = `{
			ldap_server_id                         = "-1"
			name                                   = "All Directory Service Users"
			enterprise_enrollment_enabled          = true
			personal_enrollment_enabled            = false
			account_driven_user_enrollment_enabled = true
		}`

	envElem := fmt.Sprintf(`{
			ldap_server_id                         = %q
			name                                   = %q
			enterprise_enrollment_enabled          = true
			personal_enrollment_enabled            = false
			account_driven_user_enrollment_enabled = false
		}`, ldapServerID, groupName)

	withEnvGroup := fmt.Sprintf(`
		resource "jamfplatform_pro_user_initiated_enrollment_settings" "test" {
			access_group = [%s, %s]
		}
	`, builtinElem, envElem)

	builtinOnly := fmt.Sprintf(`
		resource "jamfplatform_pro_user_initiated_enrollment_settings" "test" {
			access_group = [%s]
		}
	`, builtinElem)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             checkUIEStillExists(t),
		Steps: []resource.TestStep{
			{
				// ADD: built-in + env group.
				Config: withEnvGroup,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(uieResourceAddr, "access_group.#", "2"),
					resource.TestCheckTypeSetElemNestedAttrs(uieResourceAddr, "access_group.*", map[string]string{
						"ldap_server_id": ldapServerID,
						"name":           groupName,
					}),
					// The provider resolved the directory group id from the name;
					// assert an id came back (non-empty, not the built-in "-1"),
					// without pinning a tenant-specific value.
					checkAccessGroupResolvedID(uieResourceAddr, groupName),
					resource.TestCheckTypeSetElemNestedAttrs(uieResourceAddr, "access_group.*", map[string]string{
						"directory_service_group_id": "-1",
						"ldap_server_id":             "-1",
					}),
				),
			},
			{
				// REMOVE: built-in only.
				Config: builtinOnly,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(uieResourceAddr, "access_group.#", "1"),
					resource.TestCheckTypeSetElemNestedAttrs(uieResourceAddr, "access_group.*", map[string]string{
						"directory_service_group_id": "-1",
						"ldap_server_id":             "-1",
					}),
				),
			},
		},
	})
}

// TestAccResource_ProUserInitiatedEnrollmentSettings_CoexistenceWithReEnrollment
// is the critical correctness test. Both the uie (/v4) and re_enrollment (/v1)
// resources are managed in one config over a shared backing store. Step 1
// applies both; step 2 mutates one uie scalar (launch_self_service) AND one
// re_enrollment field (clear_policy_logs) in the same apply and asserts BOTH new
// values stick — proving the /v4 read-merge-write does not clobber the /v1
// fields and the /v1 full-replace does not clobber uie fields. The final
// PlanOnly step proves idempotency (no perpetual diff) under the shared store.
func TestAccResource_ProUserInitiatedEnrollmentSettings_CoexistenceWithReEnrollment(t *testing.T) {
	testhelpers.AccPreCheck(t)

	const reAddr = "jamfplatform_pro_re_enrollment_settings.test"

	cfg := func(launchSelfService, clearPolicyLogs bool) string {
		return fmt.Sprintf(`
			resource "jamfplatform_pro_user_initiated_enrollment_settings" "test" {
				skip_certificate_installation = false
				restrict_reenrollment         = false
				enable_computer_enrollment    = true
				create_management_account     = true
				management_username           = "tf-acc-coexist"
				launch_self_service           = %[1]t
			}

			resource "jamfplatform_pro_re_enrollment_settings" "test" {
				clear_policy_logs                  = %[2]t
				clear_location_information         = true
				clear_location_information_history = true
				clear_extension_attributes         = true
				clear_software_update_plans        = true
				clear_management_history           = "DELETE_EVERYTHING"
			}
		`, launchSelfService, clearPolicyLogs)
	}

	coexistChecks := func(launchSelfService, clearPolicyLogs bool) resource.TestCheckFunc {
		return resource.ComposeAggregateTestCheckFunc(
			resource.TestCheckResourceAttr(uieResourceAddr, "launch_self_service", fmt.Sprintf("%t", launchSelfService)),
			resource.TestCheckResourceAttr(uieResourceAddr, "management_username", "tf-acc-coexist"),
			resource.TestCheckResourceAttr(reAddr, "clear_policy_logs", fmt.Sprintf("%t", clearPolicyLogs)),
			resource.TestCheckResourceAttr(reAddr, "clear_management_history", "DELETE_EVERYTHING"),
		)
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy: resource.ComposeAggregateTestCheckFunc(
			checkUIEStillExists(t),
			checkReEnrollmentStillExistsUIE(t),
		),
		Steps: []resource.TestStep{
			{
				Config: cfg(true, true),
				Check:  coexistChecks(true, true),
			},
			{
				// Mutate one field on each resource in the same apply.
				Config: cfg(false, false),
				Check:  coexistChecks(false, false),
			},
			{
				// Idempotency: re-plan the step-2 config; expect no diff.
				Config:             cfg(false, false),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

// TestAccDataSource_ProUserInitiatedEnrollmentSettings_Basic applies the
// resource then reads it back through the data source.
func TestAccDataSource_ProUserInitiatedEnrollmentSettings_Basic(t *testing.T) {
	testhelpers.AccPreCheck(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             checkUIEStillExists(t),
		Steps: []resource.TestStep{
			{
				Config: uieScalarConfig(true, "tf-acc-ds") + `
					data "jamfplatform_pro_user_initiated_enrollment_settings" "ds" {
						depends_on = [jamfplatform_pro_user_initiated_enrollment_settings.test]
					}
				`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.jamfplatform_pro_user_initiated_enrollment_settings.ds", "id", "singleton"),
					resource.TestCheckResourceAttr("data.jamfplatform_pro_user_initiated_enrollment_settings.ds", "management_username", "tf-acc-ds"),
					resource.TestCheckResourceAttr("data.jamfplatform_pro_user_initiated_enrollment_settings.ds", "launch_self_service", "true"),
				),
			},
		},
	})
}
