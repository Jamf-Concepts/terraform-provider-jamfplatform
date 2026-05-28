//go:build acceptance

// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package computer_prestage_enrollment_test

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/testhelpers"
)

const (
	resourceName  = "jamfplatform_pro_computer_prestage_enrollment.test"
	adeFixtureRef = "jamfplatform_pro_automated_device_enrollment.fixture.id"
)

// requireADETokenBlob skips the test when JAMFPLATFORM_ADE_TOKEN is not set.
// The env var must hold the base64-encoded `.p7m` server token downloaded
// from Apple Business Manager / Apple School Manager — the same token blob
// the sibling jamfplatform_pro_automated_device_enrollment acc tests
// consume. Each test creates an ADE fixture from this blob, builds a
// prestage that depends on it, and lets TF destroy both at the end.
func requireADETokenBlob(t *testing.T) string {
	t.Helper()
	v := os.Getenv("JAMFPLATFORM_ADE_TOKEN")
	if v == "" {
		t.Skip("JAMFPLATFORM_ADE_TOKEN not set — acc tests upload a real ADE server token (.p7m base64) and create an ADE fixture before exercising the prestage.")
	}
	return v
}

// requireADESerialFixture skips the test when JAMFPLATFORM_ADE_SERIAL is
// not set. Holds a real device serial number bound to the uploaded ADE
// token; without it scope writes return DEVICE_DOES_NOT_EXIST_ON_TOKEN.
func requireADESerialFixture(t *testing.T) string {
	t.Helper()
	v := os.Getenv("JAMFPLATFORM_ADE_SERIAL")
	if v == "" {
		t.Skip("JAMFPLATFORM_ADE_SERIAL not set — scope acc test requires a real ADE-bound device serial.")
	}
	return v
}

// adeFixtureBlock emits a Terraform resource block that creates an ADE
// instance from the supplied base64 token. Returned HCL is concatenated
// with the prestage block in each test config.
func adeFixtureBlock(suffix, tokenB64 string) string {
	return fmt.Sprintf(`
resource "jamfplatform_pro_automated_device_enrollment" "fixture" {
  name                    = %q
  server_token            = %q
  server_token_wo_version = 1
}
`, "tf-acc-prestage-ade-"+suffix, tokenB64)
}

// importStateVerifyIgnore mirrors spike §13 — WriteOnly secrets and their
// `_wo_version` rotation triggers are never echoed back by Jamf Pro.
var importStateVerifyIgnore = []string{
	"timeouts",
	"recovery_lock_password",
	"recovery_lock_password_wo_version",
	"account_settings.admin_password",
	"account_settings.admin_password_wo_version",
}

// --- §13 #1: Minimal Create + Import + simple Update --------------------------

func TestAccResource_ProComputerPrestageEnrollment_Minimal(t *testing.T) {
	testhelpers.AccPreCheck(t)
	token := requireADETokenBlob(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-computer-prestage-minimal-" + suffix

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: adeFixtureBlock(suffix, token) + testAccComputerPrestageMinimalConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "display_name", name),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "profile_uuid"),
					resource.TestCheckResourceAttrSet(resourceName, "device_enrollment_program_instance_id"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: importStateVerifyIgnore,
			},
			{
				Config: adeFixtureBlock(suffix, token) + testAccComputerPrestageMinimalConfig(name+"-renamed"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "display_name", name+"-renamed"),
				),
			},
		},
	})
}

// --- §13 #2: Full Update round-trip -----------------------------------------

func TestAccResource_ProComputerPrestageEnrollment_Full_UpdateRoundTrip(t *testing.T) {
	testhelpers.AccPreCheck(t)
	token := requireADETokenBlob(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-computer-prestage-full-" + suffix

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: adeFixtureBlock(suffix, token) + testAccComputerPrestageFullConfigV1(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "display_name", name),
					resource.TestCheckResourceAttr(resourceName, "support_phone_number", "+44-1-555-0100"),
					resource.TestCheckResourceAttr(resourceName, "skip_setup_items.filevault", "true"),
					resource.TestCheckResourceAttr(resourceName, "location_information.username", "tf-acc-user"),
					resource.TestCheckResourceAttr(resourceName, "purchasing_information.apple_care_id", "AC-1"),
					resource.TestCheckResourceAttr(resourceName, "account_settings.payload_configured", "true"),
					resource.TestCheckResourceAttr(resourceName, "account_settings.admin_username", "ladmin1"),
				),
			},
			{
				Config: adeFixtureBlock(suffix, token) + testAccComputerPrestageFullConfigV2(name+"-v2"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "display_name", name+"-v2"),
					resource.TestCheckResourceAttr(resourceName, "support_phone_number", "+44-1-555-0200"),
					resource.TestCheckResourceAttr(resourceName, "skip_setup_items.filevault", "false"),
					resource.TestCheckResourceAttr(resourceName, "skip_setup_items.icloud_storage", "true"),
					resource.TestCheckResourceAttr(resourceName, "location_information.username", "tf-acc-user-v2"),
					resource.TestCheckResourceAttr(resourceName, "purchasing_information.life_expectancy", "5"),
					resource.TestCheckResourceAttr(resourceName, "account_settings.admin_username", "ladmin2"),
					resource.TestCheckResourceAttr(resourceName, "account_settings.admin_password_wo_version", "2"),
				),
			},
		},
	})
}

// --- §13 #3 / #4: Recovery Lock mutually-exclusive shapes -------------------

func TestAccResource_ProComputerPrestageEnrollment_RecoveryLockManual(t *testing.T) {
	testhelpers.AccPreCheck(t)
	token := requireADETokenBlob(t)
	suffix := testhelpers.RunSuffix()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: adeFixtureBlock(suffix, token) +
					testAccComputerPrestageRecoveryLockManualConfig("tf-acc-rl-manual-"+suffix),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "enable_recovery_lock", "true"),
					resource.TestCheckResourceAttr(resourceName, "recovery_lock_password_type", "MANUAL"),
				),
			},
		},
	})
}

func TestAccResource_ProComputerPrestageEnrollment_RecoveryLockRandom(t *testing.T) {
	testhelpers.AccPreCheck(t)
	token := requireADETokenBlob(t)
	suffix := testhelpers.RunSuffix()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: adeFixtureBlock(suffix, token) +
					testAccComputerPrestageRecoveryLockRandomConfig("tf-acc-rl-random-"+suffix),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "enable_recovery_lock", "true"),
					resource.TestCheckResourceAttr(resourceName, "recovery_lock_password_type", "RANDOM"),
				),
			},
		},
	})
}

// --- §13 #5 / #6: AccountSettings prefill modes -----------------------------

func TestAccResource_ProComputerPrestageEnrollment_AccountSettingsPrefillCustom(t *testing.T) {
	testhelpers.AccPreCheck(t)
	token := requireADETokenBlob(t)
	suffix := testhelpers.RunSuffix()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: adeFixtureBlock(suffix, token) +
					testAccComputerPrestagePrefillCustomConfig("tf-acc-prefill-custom-"+suffix),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "account_settings.prefill_type", "CUSTOM"),
					resource.TestCheckResourceAttr(resourceName, "account_settings.prefill_account_full_name", "Acc Test"),
					resource.TestCheckResourceAttr(resourceName, "account_settings.prefill_account_user_name", "acctest"),
				),
			},
		},
	})
}

func TestAccResource_ProComputerPrestageEnrollment_AccountSettingsPrefillDeviceOwner(t *testing.T) {
	testhelpers.AccPreCheck(t)
	token := requireADETokenBlob(t)
	suffix := testhelpers.RunSuffix()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: adeFixtureBlock(suffix, token) +
					testAccComputerPrestagePrefillDeviceOwnerConfig("tf-acc-prefill-device-owner-"+suffix),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "account_settings.prefill_type", "DEVICE_OWNER"),
				),
			},
		},
	})
}

// --- §13 #7: PSSO disabled (default) -----------------------------------------

func TestAccResource_ProComputerPrestageEnrollment_NoPSSOFields(t *testing.T) {
	testhelpers.AccPreCheck(t)
	token := requireADETokenBlob(t)
	suffix := testhelpers.RunSuffix()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: adeFixtureBlock(suffix, token) +
					testAccComputerPrestageMinimalConfig("tf-acc-no-psso-"+suffix),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "psso_enabled", "false"),
				),
			},
		},
	})
}

// --- §13 #8: Scope round-trip ------------------------------------------------

func TestAccResource_ProComputerPrestageEnrollment_ScopeAssignments(t *testing.T) {
	testhelpers.AccPreCheck(t)
	token := requireADETokenBlob(t)
	serial := requireADESerialFixture(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-scope-" + suffix

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: adeFixtureBlock(suffix, token) + testAccComputerPrestageScopeConfig(name, ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "scope_serial_numbers.#", "0"),
				),
			},
			{
				Config: adeFixtureBlock(suffix, token) + testAccComputerPrestageScopeConfig(name, serial),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "scope_serial_numbers.#", "1"),
				),
			},
			{
				Config: adeFixtureBlock(suffix, token) + testAccComputerPrestageScopeConfig(name, ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "scope_serial_numbers.#", "0"),
				),
			},
		},
	})
}

// --- §13 #9: anchor_certificates silent-rollback hard-error path ------------

func TestAccResource_ProComputerPrestageEnrollment_AnchorCertificatesRollback(t *testing.T) {
	testhelpers.AccPreCheck(t)
	token := requireADETokenBlob(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-anchor-" + suffix

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: adeFixtureBlock(suffix, token) + testAccComputerPrestageMinimalConfig(name),
				Check:  resource.TestCheckResourceAttrSet(resourceName, "id"),
			},
			{
				Config:      adeFixtureBlock(suffix, token) + testAccComputerPrestageAnchorsConfig(name, `["dGVzdA=="]`),
				ExpectError: regexp.MustCompile(`did not commit|anchor_certificates`),
			},
		},
	})
}

// --- §13 #10: ExpectError per declared validator ---------------------------

func TestAccResource_ProComputerPrestageEnrollment_ExpectError_RandomConflictsWithPassword(t *testing.T) {
	testhelpers.AccPreCheck(t)
	token := requireADETokenBlob(t)
	suffix := testhelpers.RunSuffix()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: adeFixtureBlock(suffix, token) + fmt.Sprintf(`
resource "jamfplatform_pro_computer_prestage_enrollment" "test" {
  display_name                          = "tf-acc-expect-random-with-pwd-%s"
  mandatory                             = true
  mdm_removable                         = true
  require_authentication                = false
  device_enrollment_program_instance_id = %s
  keep_existing_location_information    = false
  keep_existing_site_membership         = false
  auto_advance_setup                    = false
  install_profiles_during_setup         = false
  prevent_activation_lock               = false
  enable_device_based_activation_lock   = false

  enable_recovery_lock              = true
  recovery_lock_password_type       = "RANDOM"
  recovery_lock_password            = "conflict-with-RANDOM"
  recovery_lock_password_wo_version = 1
}
`, suffix, adeFixtureRef),
				ExpectError: regexp.MustCompile(`conflicts with recovery_lock_password_type = RANDOM`),
			},
		},
	})
}

func TestAccResource_ProComputerPrestageEnrollment_ExpectError_PasswordRequiresEnable(t *testing.T) {
	testhelpers.AccPreCheck(t)
	token := requireADETokenBlob(t)
	suffix := testhelpers.RunSuffix()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: adeFixtureBlock(suffix, token) + fmt.Sprintf(`
resource "jamfplatform_pro_computer_prestage_enrollment" "test" {
  display_name                          = "tf-acc-expect-pwd-disabled-%s"
  mandatory                             = true
  mdm_removable                         = true
  require_authentication                = false
  device_enrollment_program_instance_id = %s
  keep_existing_location_information    = false
  keep_existing_site_membership         = false
  auto_advance_setup                    = false
  install_profiles_during_setup         = false
  prevent_activation_lock               = false
  enable_device_based_activation_lock   = false

  enable_recovery_lock              = false
  recovery_lock_password_type       = "MANUAL"
  recovery_lock_password            = "ShouldFailHere"
  recovery_lock_password_wo_version = 1
}
`, suffix, adeFixtureRef),
				ExpectError: regexp.MustCompile(`enable_recovery_lock = true`),
			},
		},
	})
}

func TestAccResource_ProComputerPrestageEnrollment_ExpectError_PrefillCustomRequiresNames(t *testing.T) {
	testhelpers.AccPreCheck(t)
	token := requireADETokenBlob(t)
	suffix := testhelpers.RunSuffix()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: adeFixtureBlock(suffix, token) + fmt.Sprintf(`
resource "jamfplatform_pro_computer_prestage_enrollment" "test" {
  display_name                          = "tf-acc-expect-prefill-custom-missing-names-%s"
  mandatory                             = true
  mdm_removable                         = true
  require_authentication                = false
  device_enrollment_program_instance_id = %s
  keep_existing_location_information    = false
  keep_existing_site_membership         = false
  auto_advance_setup                    = false
  install_profiles_during_setup         = false
  prevent_activation_lock               = false
  enable_device_based_activation_lock   = false

  account_settings = {
    payload_configured = true
    prefill_type       = "CUSTOM"
  }
}
`, suffix, adeFixtureRef),
				ExpectError: regexp.MustCompile(`prefill_account_(full|user)_name is required`),
			},
		},
	})
}

func TestAccResource_ProComputerPrestageEnrollment_ExpectError_BadRecoveryLockType(t *testing.T) {
	testhelpers.AccPreCheck(t)
	token := requireADETokenBlob(t)
	suffix := testhelpers.RunSuffix()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: adeFixtureBlock(suffix, token) + fmt.Sprintf(`
resource "jamfplatform_pro_computer_prestage_enrollment" "test" {
  display_name                          = "tf-acc-expect-bad-rl-type-%s"
  mandatory                             = true
  mdm_removable                         = true
  require_authentication                = false
  device_enrollment_program_instance_id = %s
  keep_existing_location_information    = false
  keep_existing_site_membership         = false
  auto_advance_setup                    = false
  install_profiles_during_setup         = false
  prevent_activation_lock               = false
  enable_device_based_activation_lock   = false

  recovery_lock_password_type = "BOGUS_VALUE"
}
`, suffix, adeFixtureRef),
				ExpectError: regexp.MustCompile(`must be one of: \["MANUAL" "RANDOM"\]`),
			},
		},
	})
}

func TestAccResource_ProComputerPrestageEnrollment_ExpectError_BadPrefillType(t *testing.T) {
	testhelpers.AccPreCheck(t)
	token := requireADETokenBlob(t)
	suffix := testhelpers.RunSuffix()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: adeFixtureBlock(suffix, token) + fmt.Sprintf(`
resource "jamfplatform_pro_computer_prestage_enrollment" "test" {
  display_name                          = "tf-acc-expect-bad-prefill-%s"
  mandatory                             = true
  mdm_removable                         = true
  require_authentication                = false
  device_enrollment_program_instance_id = %s
  keep_existing_location_information    = false
  keep_existing_site_membership         = false
  auto_advance_setup                    = false
  install_profiles_during_setup         = false
  prevent_activation_lock               = false
  enable_device_based_activation_lock   = false

  account_settings = {
    prefill_type = "UNKNOWN"
  }
}
`, suffix, adeFixtureRef),
				ExpectError: regexp.MustCompile(`must be one of: \["CUSTOM" "DEVICE_OWNER"\]`),
			},
		},
	})
}

func TestAccResource_ProComputerPrestageEnrollment_ExpectError_BadUserAccountType(t *testing.T) {
	testhelpers.AccPreCheck(t)
	token := requireADETokenBlob(t)
	suffix := testhelpers.RunSuffix()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: adeFixtureBlock(suffix, token) + fmt.Sprintf(`
resource "jamfplatform_pro_computer_prestage_enrollment" "test" {
  display_name                          = "tf-acc-expect-bad-uat-%s"
  mandatory                             = true
  mdm_removable                         = true
  require_authentication                = false
  device_enrollment_program_instance_id = %s
  keep_existing_location_information    = false
  keep_existing_site_membership         = false
  auto_advance_setup                    = false
  install_profiles_during_setup         = false
  prevent_activation_lock               = false
  enable_device_based_activation_lock   = false

  account_settings = {
    user_account_type = "ROOT"
  }
}
`, suffix, adeFixtureRef),
				ExpectError: regexp.MustCompile(`must be one of: \["ADMINISTRATOR" "STANDARD" "SKIP"\]`),
			},
		},
	})
}

func TestAccResource_ProComputerPrestageEnrollment_ExpectError_BadMinOsTarget(t *testing.T) {
	testhelpers.AccPreCheck(t)
	token := requireADETokenBlob(t)
	suffix := testhelpers.RunSuffix()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: adeFixtureBlock(suffix, token) + fmt.Sprintf(`
resource "jamfplatform_pro_computer_prestage_enrollment" "test" {
  display_name                          = "tf-acc-expect-bad-min-os-%s"
  mandatory                             = true
  mdm_removable                         = true
  require_authentication                = false
  device_enrollment_program_instance_id = %s
  keep_existing_location_information    = false
  keep_existing_site_membership         = false
  auto_advance_setup                    = false
  install_profiles_during_setup         = false
  prevent_activation_lock               = false
  enable_device_based_activation_lock   = false

  prestage_minimum_os_target_version_type = "LATEST_ONLY"
}
`, suffix, adeFixtureRef),
				ExpectError: regexp.MustCompile(`must be one of`),
			},
		},
	})
}

// --- Config helpers (prestage block only — ADE fixture is concatenated by caller) ---

func testAccComputerPrestageMinimalConfig(name string) string {
	return fmt.Sprintf(`
resource "jamfplatform_pro_computer_prestage_enrollment" "test" {
  display_name                          = %q
  mandatory                             = true
  mdm_removable                         = true
  require_authentication                = false
  device_enrollment_program_instance_id = %s
  keep_existing_location_information    = false
  keep_existing_site_membership         = false
  auto_advance_setup                    = false
  install_profiles_during_setup         = false
  prevent_activation_lock               = false
  enable_device_based_activation_lock   = false

  location_information   = {}
  purchasing_information = {}
  account_settings       = {}
  skip_setup_items       = {}

  scope_serial_numbers = []
}
`, name, adeFixtureRef)
}

func testAccComputerPrestageFullConfigV1(name string) string {
	return fmt.Sprintf(`
resource "jamfplatform_pro_computer_prestage_enrollment" "test" {
  display_name                          = %q
  mandatory                             = true
  mdm_removable                         = true
  default_prestage                      = false
  support_phone_number                  = "+44-1-555-0100"
  support_email_address                 = "ops@example.test"
  department                            = "Operations"
  require_authentication                = true
  authentication_prompt                 = "Welcome to acceptance"
  device_enrollment_program_instance_id = %s
  keep_existing_location_information    = false
  keep_existing_site_membership         = false
  auto_advance_setup                    = false
  install_profiles_during_setup         = false
  prevent_activation_lock               = false
  enable_device_based_activation_lock   = false

  skip_setup_items = {
    filevault      = true
    siri           = true
    icloud_storage = false
  }

  location_information = {
    username = "tf-acc-user"
    email    = "tf-acc@example.test"
  }

  purchasing_information = {
    purchased       = true
    apple_care_id   = "AC-1"
    life_expectancy = 3
  }

  account_settings = {
    payload_configured                           = true
    local_admin_account_enabled                  = true
    admin_username                               = "ladmin1"
    admin_password                               = "InitialP@ss1"
    admin_password_wo_version                    = 1
    user_account_type                            = "ADMINISTRATOR"
    prefill_primary_account_info_feature_enabled = true
    prefill_type                                 = "DEVICE_OWNER"
  }

  scope_serial_numbers = []
}
`, name, adeFixtureRef)
}

func testAccComputerPrestageFullConfigV2(name string) string {
	return fmt.Sprintf(`
resource "jamfplatform_pro_computer_prestage_enrollment" "test" {
  display_name                          = %q
  mandatory                             = false
  mdm_removable                         = false
  default_prestage                      = false
  support_phone_number                  = "+44-1-555-0200"
  support_email_address                 = "newops@example.test"
  department                            = "NewOps"
  require_authentication                = false
  authentication_prompt                 = "Updated prompt"
  device_enrollment_program_instance_id = %s
  keep_existing_location_information    = true
  keep_existing_site_membership         = true
  auto_advance_setup                    = true
  install_profiles_during_setup         = false
  prevent_activation_lock               = true
  enable_device_based_activation_lock   = false

  skip_setup_items = {
    filevault                   = false
    siri                        = false
    icloud_storage              = true
    additional_privacy_settings = true
  }

  location_information = {
    username = "tf-acc-user-v2"
    realname = "Updated User"
    email    = "tf-acc-v2@example.test"
  }

  purchasing_information = {
    purchased       = true
    apple_care_id   = "AC-2"
    life_expectancy = 5
    vendor          = "Acme"
  }

  account_settings = {
    payload_configured                           = true
    local_admin_account_enabled                  = true
    admin_username                               = "ladmin2"
    admin_password                               = "RotatedP@ss2"
    admin_password_wo_version                    = 2
    user_account_type                            = "ADMINISTRATOR"
    prefill_primary_account_info_feature_enabled = true
    prefill_type                                 = "DEVICE_OWNER"
  }

  custom_package_ids = []

  scope_serial_numbers = []
}
`, name, adeFixtureRef)
}

func testAccComputerPrestageRecoveryLockManualConfig(name string) string {
	return fmt.Sprintf(`
resource "jamfplatform_pro_computer_prestage_enrollment" "test" {
  display_name                          = %q
  mandatory                             = true
  mdm_removable                         = true
  require_authentication                = false
  device_enrollment_program_instance_id = %s
  keep_existing_location_information    = false
  keep_existing_site_membership         = false
  auto_advance_setup                    = false
  install_profiles_during_setup         = false
  prevent_activation_lock               = false
  enable_device_based_activation_lock   = false

  enable_recovery_lock              = true
  recovery_lock_password_type       = "MANUAL"
  recovery_lock_password            = "RecoveryP@ss123"
  recovery_lock_password_wo_version = 1

  location_information   = {}
  purchasing_information = {}
  account_settings       = {}
}
`, name, adeFixtureRef)
}

func testAccComputerPrestageRecoveryLockRandomConfig(name string) string {
	return fmt.Sprintf(`
resource "jamfplatform_pro_computer_prestage_enrollment" "test" {
  display_name                          = %q
  mandatory                             = true
  mdm_removable                         = true
  require_authentication                = false
  device_enrollment_program_instance_id = %s
  keep_existing_location_information    = false
  keep_existing_site_membership         = false
  auto_advance_setup                    = false
  install_profiles_during_setup         = false
  prevent_activation_lock               = false
  enable_device_based_activation_lock   = false

  enable_recovery_lock        = true
  recovery_lock_password_type = "RANDOM"

  location_information   = {}
  purchasing_information = {}
  account_settings       = {}
}
`, name, adeFixtureRef)
}

func testAccComputerPrestagePrefillCustomConfig(name string) string {
	return fmt.Sprintf(`
resource "jamfplatform_pro_computer_prestage_enrollment" "test" {
  display_name                          = %q
  mandatory                             = true
  mdm_removable                         = true
  require_authentication                = false
  device_enrollment_program_instance_id = %s
  keep_existing_location_information    = false
  keep_existing_site_membership         = false
  auto_advance_setup                    = false
  install_profiles_during_setup         = false
  prevent_activation_lock               = false
  enable_device_based_activation_lock   = false

  account_settings = {
    payload_configured                           = true
    prefill_primary_account_info_feature_enabled = true
    prefill_type                                 = "CUSTOM"
    prefill_account_full_name                    = "Acc Test"
    prefill_account_user_name                    = "acctest"
  }

  location_information   = {}
  purchasing_information = {}
}
`, name, adeFixtureRef)
}

func testAccComputerPrestagePrefillDeviceOwnerConfig(name string) string {
	return fmt.Sprintf(`
resource "jamfplatform_pro_computer_prestage_enrollment" "test" {
  display_name                          = %q
  mandatory                             = true
  mdm_removable                         = true
  require_authentication                = false
  device_enrollment_program_instance_id = %s
  keep_existing_location_information    = false
  keep_existing_site_membership         = false
  auto_advance_setup                    = false
  install_profiles_during_setup         = false
  prevent_activation_lock               = false
  enable_device_based_activation_lock   = false

  account_settings = {
    payload_configured                           = true
    prefill_primary_account_info_feature_enabled = true
    prefill_type                                 = "DEVICE_OWNER"
  }

  location_information   = {}
  purchasing_information = {}
}
`, name, adeFixtureRef)
}

func testAccComputerPrestageScopeConfig(name, serial string) string {
	scope := "[]"
	if serial != "" {
		scope = fmt.Sprintf(`[%q]`, serial)
	}
	return fmt.Sprintf(`
resource "jamfplatform_pro_computer_prestage_enrollment" "test" {
  display_name                          = %q
  mandatory                             = true
  mdm_removable                         = true
  require_authentication                = false
  device_enrollment_program_instance_id = %s
  keep_existing_location_information    = false
  keep_existing_site_membership         = false
  auto_advance_setup                    = false
  install_profiles_during_setup         = false
  prevent_activation_lock               = false
  enable_device_based_activation_lock   = false

  location_information   = {}
  purchasing_information = {}
  account_settings       = {}

  scope_serial_numbers = %s
}
`, name, adeFixtureRef, scope)
}

func testAccComputerPrestageAnchorsConfig(name, anchors string) string {
	return fmt.Sprintf(`
resource "jamfplatform_pro_computer_prestage_enrollment" "test" {
  display_name                          = %q
  mandatory                             = true
  mdm_removable                         = true
  require_authentication                = false
  device_enrollment_program_instance_id = %s
  keep_existing_location_information    = false
  keep_existing_site_membership         = false
  auto_advance_setup                    = false
  install_profiles_during_setup         = false
  prevent_activation_lock               = false
  enable_device_based_activation_lock   = false

  anchor_certificates = %s

  location_information   = {}
  purchasing_information = {}
  account_settings       = {}
}
`, name, adeFixtureRef, anchors)
}
