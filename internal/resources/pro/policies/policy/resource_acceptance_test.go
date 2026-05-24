// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

// Tests in this file talk to the Jamf ProClassic /policies endpoint. Classic
// has known concurrency issues when multiple writes hit the same resource
// type — keep these tests serial with any other classic acceptance work.

package policy_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/proclassic"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/testhelpers"
)

// packageFixturePath resolves the shared jamf-cli .pkg fixture committed under
// internal/resources/pro/inventory/package/test_fixtures/. The path is computed
// relative to this test file so the result is invariant to the working
// directory `go test` was invoked from.
func packageFixturePath(t *testing.T, name string) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("could not resolve caller path for fixture lookup")
	}
	dir := filepath.Dir(file)
	abs, err := filepath.Abs(filepath.Join(dir, "..", "..", "inventory", "package", "test_fixtures", name))
	if err != nil {
		t.Fatalf("resolving fixture path %q: %v", name, err)
	}
	if _, err := os.Stat(abs); err != nil {
		t.Fatalf("fixture %q not present at %q: %v", name, abs, err)
	}
	return abs
}

// testAccCheckPolicyDestroy verifies policies created during the test were
// destroyed.
func testAccCheckPolicyDestroy(t *testing.T) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		c := proclassic.New(testhelpers.NewAcceptanceClient(t))
		ctx := context.Background()

		for _, rs := range s.RootModule().Resources {
			if rs.Type != "jamfplatform_pro_policy" {
				continue
			}
			_, err := c.GetPolicyByID(ctx, rs.Primary.ID)
			if err != nil {
				if helpers.IsNotFoundError(err) {
					continue
				}
				return fmt.Errorf("error checking Jamf Pro policy %s: %s", rs.Primary.ID, err)
			}
			return fmt.Errorf("Jamf Pro policy %s still exists", rs.Primary.ID)
		}
		return nil
	}
}

func policyConfigMinimal(name string) string {
	return fmt.Sprintf(`
resource "jamfplatform_pro_policy" "test" {
  general = {
    name    = %q
    enabled = true
  }
}
`, name)
}

func policyConfigEnabledFlip(name string) string {
	return fmt.Sprintf(`
resource "jamfplatform_pro_policy" "test" {
  general = {
    name    = %q
    enabled = false
  }
}
`, name)
}

func policyConfigSelfService(name string) string {
	return fmt.Sprintf(`
resource "jamfplatform_pro_policy" "test" {
  general = {
    name    = %q
    enabled = true
  }
  self_service = {
    use_for_self_service      = true
    self_service_display_name = %q
    notification_enabled      = true
    notification_type       = "Self Service"
    notification_subject      = "tf-acc"
    notification_message      = "Test policy"
  }
}
`, name, name)
}

func policyConfigAllComputers(name string) string {
	return fmt.Sprintf(`
resource "jamfplatform_pro_policy" "test" {
  general = {
    name = %q
  }
  scope = {
    all_computers = true
  }
}
`, name)
}

// TestAccPolicyResource_Minimal covers the smallest viable policy: name only.
// Verifies Create/Read/Update/Delete + import.
func TestAccPolicyResource_Minimal(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-policy-min-" + suffix
	renamed := name + "-renamed"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPolicyDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: policyConfigMinimal(name),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("general").AtMapKey("name"),
						knownvalue.StringExact(name),
					),
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("general").AtMapKey("enabled"),
						knownvalue.Bool(true),
					),
				},
			},
			{
				Config: policyConfigMinimal(renamed),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("general").AtMapKey("name"),
						knownvalue.StringExact(renamed),
					),
				},
			},
			{
				Config: policyConfigEnabledFlip(renamed),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("general").AtMapKey("enabled"),
						knownvalue.Bool(false),
					),
				},
			},
			{
				ResourceName:                         "jamfplatform_pro_policy.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "id",
			},
		},
	})
}

// TestAccPolicyResource_SelfServiceNotificationSplit confirms the two
// <notification> elements round-trip cleanly through the schema's split
// notification_enabled + notification_type attributes.
func TestAccPolicyResource_SelfServiceNotificationSplit(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-policy-ss-" + suffix

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPolicyDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: policyConfigSelfService(name),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("self_service").AtMapKey("notification_enabled"),
						knownvalue.Bool(true),
					),
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("self_service").AtMapKey("notification_type"),
						knownvalue.StringExact("Self Service"),
					),
				},
			},
		},
	})
}

// TestAccPolicyResource_AllComputers verifies all_computers=true creates
// without per-computer targets.
func TestAccPolicyResource_AllComputers(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-policy-all-" + suffix

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPolicyDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: policyConfigAllComputers(name),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("scope").AtMapKey("all_computers"),
						knownvalue.Bool(true),
					),
				},
			},
		},
	})
}

// policyConfigGeneralFull exercises every general top-level attribute the
// schema exposes (excluding category/site reference IDs, which use -1 to
// pin "NONE" so the test is independent of tenant inventory).
func policyConfigGeneralFull(name string) string {
	return fmt.Sprintf(`
resource "jamfplatform_pro_policy" "test" {
  general = {
    name                          = %q
    enabled                       = true
    trigger                       = "EVENT"
    trigger_checkin               = true
    trigger_enrollment_complete   = true
    trigger_login                 = true
    trigger_logout                = false
    trigger_network_state_changed = true
    trigger_startup               = true
    trigger_other                 = "tf-acc-event"
    frequency                     = "Once per computer"
    retry_event                   = "check-in"
    retry_attempts                = 3
    notify_on_each_failed_retry   = true
    location_user_only            = false
    target_drive                  = "/"
    offline                       = false
    category_id                   = "-1"
    site_id                       = "-1"
  }
}
`, name)
}

// policyConfigGeneralWithSubBlocks layers the three nested sub-blocks
// (date_time_limitations, network_limitations, override_default_settings)
// on top of the full general block. Retry attributes are explicitly cleared
// because Jamf Pro rejects any retry configuration when frequency is not
// "Once per computer", and Optional+Computed values otherwise carry over
// from the prior step via UseStateForUnknown.
func policyConfigGeneralWithSubBlocks(name string) string {
	return fmt.Sprintf(`
resource "jamfplatform_pro_policy" "test" {
  general = {
    name           = %q
    enabled        = true
    frequency      = "Ongoing"
    retry_event    = "none"
    retry_attempts = -1
    category_id    = "-1"
    site_id        = "-1"

    date_time_limitations = {
      activation_date  = "2026-01-01 01:00:00"
      expiration_date  = "2027-01-01 01:00:00"
      no_execute_on    = ["Sun", "Sat"]
      no_execute_start = "1:00 AM"
      no_execute_end   = "2:00 AM"
    }

    network_limitations = {
      minimum_network_connection = "Ethernet"
      any_ip_address             = true
    }

    override_default_settings = {
      target_drive       = "/"
      distribution_point = "default"
      force_afp_smb      = false
      sus                = "default"
    }
  }
}
`, name)
}

// TestAccPolicyResource_GeneralFullCoverage exercises every general-section
// attribute (top-level + each nested sub-block) and confirms the wire echoes
// match what was sent. Import-state round-trip for the nested sub-blocks is
// not asserted here — by design (see state_builders.assignPolicyResourceModel)
// Optional+Computed nested sections are only populated on Read when the
// caller already manages them, so a freshly-imported state will not contain
// date_time_limitations / network_limitations / override_default_settings.
// TestAccPolicyResource_Minimal already covers import for the top-level
// general attributes.
func TestAccPolicyResource_GeneralFullCoverage(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-policy-gen-" + suffix

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPolicyDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: policyConfigGeneralFull(name),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("general").AtMapKey("trigger_checkin"),
						knownvalue.Bool(true),
					),
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("general").AtMapKey("trigger_other"),
						knownvalue.StringExact("tf-acc-event"),
					),
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("general").AtMapKey("frequency"),
						knownvalue.StringExact("Once per computer"),
					),
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("general").AtMapKey("retry_event"),
						knownvalue.StringExact("check-in"),
					),
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("general").AtMapKey("retry_attempts"),
						knownvalue.Int64Exact(3),
					),
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("general").AtMapKey("notify_on_each_failed_retry"),
						knownvalue.Bool(true),
					),
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("general").AtMapKey("target_drive"),
						knownvalue.StringExact("/"),
					),
				},
			},
			{
				Config: policyConfigGeneralWithSubBlocks(name),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("general").AtMapKey("date_time_limitations").AtMapKey("activation_date"),
						knownvalue.StringExact("2026-01-01 01:00:00"),
					),
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("general").AtMapKey("date_time_limitations").AtMapKey("expiration_date"),
						knownvalue.StringExact("2027-01-01 01:00:00"),
					),
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("general").AtMapKey("date_time_limitations").AtMapKey("no_execute_start"),
						knownvalue.StringExact("1:00 AM"),
					),
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("general").AtMapKey("network_limitations").AtMapKey("minimum_network_connection"),
						knownvalue.StringExact("Ethernet"),
					),
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("general").AtMapKey("network_limitations").AtMapKey("any_ip_address"),
						knownvalue.Bool(true),
					),
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("general").AtMapKey("override_default_settings").AtMapKey("distribution_point"),
						knownvalue.StringExact("default"),
					),
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("general").AtMapKey("override_default_settings").AtMapKey("sus"),
						knownvalue.StringExact("default"),
					),
				},
			},
		},
	})
}

// policyConfigReboot exposes every reboot-section attribute. The
// specify_startup variant is parameterised so the test can sweep all three
// validator-accepted values.
func policyConfigReboot(name, specifyStartup string) string {
	return fmt.Sprintf(`
resource "jamfplatform_pro_policy" "test" {
  general = {
    name = %q
  }
  reboot = {
    message                        = "tf-acc reboot message"
    startup_disk                   = "Current Startup Disk"
    specify_startup                = %q
    no_user_logged_in              = "Restart immediately"
    user_logged_in                 = "Restart if a package or update requires it"
    minutes_until_reboot           = 10
    start_reboot_timer_immediately = true
    file_vault_2_reboot            = false
  }
}
`, name, specifyStartup)
}

// TestAccPolicyResource_RebootFullCoverage exercises every reboot attribute
// and sweeps specify_startup through all three values the validator accepts:
// the wire-empty default, the standard restart label, and the MDM
// kernel-cache-rebuild label (per Probe #2 in PHASE_2_6_SPIKE.md).
// Import-state round-trip is not asserted — by design (see
// state_builders.assignPolicyResourceModel) the reboot section is only
// populated on Read when the caller already manages it, so a freshly-imported
// state will not contain the reboot block.
func policyConfigPackageConfigurationDistributionPoint(name, dp string) string {
	return fmt.Sprintf(`
resource "jamfplatform_pro_policy" "test" {
  general = {
    name = %q
  }
  package_configuration = {
    distribution_point = %q
  }
}
`, name, dp)
}

// TestAccPolicyResource_PackageConfigurationDistributionPoint exercises the
// top-level `<package_configuration><distribution_point>` wire field added in
// SDK 0.8.1-…46aec40edb28. The wire returns this value as a peer of <packages>
// (see PHASE_2_6_SPIKE.md §4 + Appendix). Test omits the `packages` set
// entirely — there is no clean "NONE" value for packages[].id, so per-package
// coverage is deferred to PR #5 with a real `jamfplatform_pro_package` fixture.
// Import-state round-trip is not asserted (per
// state_builders.assignPolicyResourceModel, optional sub-blocks are only
// populated on Read when already managed by the caller).
func TestAccPolicyResource_PackageConfigurationDistributionPoint(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-policy-pkgcfg-dp-" + suffix

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPolicyDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: policyConfigPackageConfigurationDistributionPoint(name, "Dummy DP"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("package_configuration").AtMapKey("distribution_point"),
						knownvalue.StringExact("Dummy DP"),
					),
				},
			},
		},
	})
}

func policyConfigPackageConfigurationPackages(name, pkgName, pkgFileName, pkgSrc, action string) string {
	return fmt.Sprintf(`
resource "jamfplatform_pro_package" "fixture" {
  display_name        = %q
  file_name           = %q
  package_file_source = %q
  info                = "tf-acc package fixture for jamfplatform_pro_policy package_configuration coverage"
}

resource "jamfplatform_pro_policy" "test" {
  general = {
    name = %q
  }
  package_configuration = {
    distribution_point = "Dummy DP"
    packages = [
      {
        id     = jamfplatform_pro_package.fixture.id
        action = %q
      },
    ]
  }
}
`, pkgName, pkgFileName, pkgSrc, name, action)
}

// TestAccPolicyResource_PackageConfigurationPackages exercises the
// `package_configuration.packages` set together with the top-level
// distribution_point. A jamfplatform_pro_package resource uploads the
// committed jamf-cli .pkg fixture (shared with the inventory/package acc
// suite) so the policy can reference a real package ID. Step 2 swaps the
// per-package action Install → Cache to exercise the Update path on the
// policy's packages set without re-uploading the package.
// Import-state round-trip is not asserted (per
// state_builders.assignPolicyResourceModel, optional sub-blocks are only
// populated on Read when already managed by the caller).
func TestAccPolicyResource_PackageConfigurationPackages(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	policyName := "tf-acc-policy-pkgcfg-pkgs-" + suffix
	pkgDisplay := "tf-acc-pkg-fixture-" + suffix
	pkgFileName := "tf-acc-pkg-fixture-" + suffix + ".pkg"
	pkgSrc := packageFixturePath(t, "jamf-cli-1.15.0.pkg")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPolicyDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: policyConfigPackageConfigurationPackages(policyName, pkgDisplay, pkgFileName, pkgSrc, "Install"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("package_configuration").AtMapKey("distribution_point"),
						knownvalue.StringExact("Dummy DP"),
					),
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("package_configuration").AtMapKey("packages"),
						knownvalue.SetSizeExact(1),
					),
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("package_configuration").AtMapKey("packages").AtSliceIndex(0).AtMapKey("action"),
						knownvalue.StringExact("Install"),
					),
				},
			},
			{
				Config: policyConfigPackageConfigurationPackages(policyName, pkgDisplay, pkgFileName, pkgSrc, "Cache"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("package_configuration").AtMapKey("packages").AtSliceIndex(0).AtMapKey("action"),
						knownvalue.StringExact("Cache"),
					),
				},
			},
		},
	})
}

func TestAccPolicyResource_RebootFullCoverage(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-policy-reboot-" + suffix

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPolicyDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: policyConfigReboot(name, ""),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("reboot").AtMapKey("specify_startup"),
						knownvalue.StringExact(""),
					),
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("reboot").AtMapKey("message"),
						knownvalue.StringExact("tf-acc reboot message"),
					),
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("reboot").AtMapKey("minutes_until_reboot"),
						knownvalue.Int64Exact(10),
					),
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("reboot").AtMapKey("start_reboot_timer_immediately"),
						knownvalue.Bool(true),
					),
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("reboot").AtMapKey("no_user_logged_in"),
						knownvalue.StringExact("Restart immediately"),
					),
				},
			},
			{
				Config: policyConfigReboot(name, "Standard Restart"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("reboot").AtMapKey("specify_startup"),
						knownvalue.StringExact("Standard Restart"),
					),
				},
			},
			{
				Config: policyConfigReboot(name, "MDM Restart with Kernel Cache Rebuild"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("reboot").AtMapKey("specify_startup"),
						knownvalue.StringExact("MDM Restart with Kernel Cache Rebuild"),
					),
				},
			},
		},
	})
}

func policyConfigScripts(policyName, scriptName, priority, scriptPolicyPriority, p4 string) string {
	return fmt.Sprintf(`
resource "jamfplatform_pro_script" "fixture" {
  name            = %q
  priority        = %q
  info            = "tf-acc script fixture for jamfplatform_pro_policy scripts coverage"
  script_contents = <<-EOT
    #!/bin/sh
    echo "tf-acc-script"
  EOT
}

resource "jamfplatform_pro_policy" "test" {
  general = {
    name = %q
  }
  scripts = {
    scripts = [
      {
        id         = jamfplatform_pro_script.fixture.id
        priority   = %q
        parameter4 = %q
      },
    ]
  }
}
`, scriptName, priority, policyName, scriptPolicyPriority, p4)
}

// TestAccPolicyResource_ScriptsFullCoverage creates a jamfplatform_pro_script
// fixture (the standalone script resource uses BEFORE/AFTER/AT_REBOOT) and
// references its ID from policy.scripts.scripts. The policy's wire form for
// `priority` is `Before`/`After`/`At Reboot` (per PHASE_2_6_SPIKE.md §5
// + Appendix), distinct from the fixture's enum. Step 2 swaps priority and
// parameter4 to exercise the Update path on the scripts set without
// recreating the fixture.
func TestAccPolicyResource_ScriptsFullCoverage(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	policyName := "tf-acc-policy-scripts-" + suffix
	scriptName := "tf-acc-script-fixture-" + suffix

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPolicyDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: policyConfigScripts(policyName, scriptName, "AFTER", "Before", "tf-acc-p4-initial"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("scripts").AtMapKey("scripts"),
						knownvalue.SetSizeExact(1),
					),
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("scripts").AtMapKey("scripts").AtSliceIndex(0).AtMapKey("priority"),
						knownvalue.StringExact("Before"),
					),
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("scripts").AtMapKey("scripts").AtSliceIndex(0).AtMapKey("parameter4"),
						knownvalue.StringExact("tf-acc-p4-initial"),
					),
				},
			},
			{
				Config: policyConfigScripts(policyName, scriptName, "AFTER", "After", "tf-acc-p4-updated"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("scripts").AtMapKey("scripts").AtSliceIndex(0).AtMapKey("priority"),
						knownvalue.StringExact("After"),
					),
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("scripts").AtMapKey("scripts").AtSliceIndex(0).AtMapKey("parameter4"),
						knownvalue.StringExact("tf-acc-p4-updated"),
					),
				},
			},
		},
	})
}

func policyConfigPrinters(policyName, printerName, action string, makeDefault, leaveExistingDefault bool) string {
	return fmt.Sprintf(`
resource "jamfplatform_pro_printer" "fixture" {
  name = %q
  uri  = "ipp://10.1.20.120/"
}

resource "jamfplatform_pro_policy" "test" {
  general = {
    name = %q
  }
  printers = {
    leave_existing_default = %t
    printers = [
      {
        id           = jamfplatform_pro_printer.fixture.id
        action       = %q
        make_default = %t
      },
    ]
  }
}
`, printerName, policyName, leaveExistingDefault, action, makeDefault)
}

// TestAccPolicyResource_PrintersFullCoverage creates a jamfplatform_pro_printer
// fixture and references it from policy.printers.printers. Step 2 swaps action
// `install` → `uninstall` and toggles make_default + leave_existing_default to
// exercise the Update path. The classic API also returns a `size` field which
// the schema exposes as Computed; not asserted explicitly because the framework
// validates Computed values are non-Unknown after apply.
func TestAccPolicyResource_PrintersFullCoverage(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	policyName := "tf-acc-policy-printers-" + suffix
	printerName := "tf-acc-printer-fixture-" + suffix

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPolicyDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: policyConfigPrinters(policyName, printerName, "install", true, false),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("printers").AtMapKey("printers"),
						knownvalue.SetSizeExact(1),
					),
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("printers").AtMapKey("printers").AtSliceIndex(0).AtMapKey("action"),
						knownvalue.StringExact("install"),
					),
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("printers").AtMapKey("printers").AtSliceIndex(0).AtMapKey("make_default"),
						knownvalue.Bool(true),
					),
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("printers").AtMapKey("leave_existing_default"),
						knownvalue.Bool(false),
					),
				},
			},
			{
				Config: policyConfigPrinters(policyName, printerName, "uninstall", false, true),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("printers").AtMapKey("printers").AtSliceIndex(0).AtMapKey("action"),
						knownvalue.StringExact("uninstall"),
					),
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("printers").AtMapKey("printers").AtSliceIndex(0).AtMapKey("make_default"),
						knownvalue.Bool(false),
					),
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("printers").AtMapKey("leave_existing_default"),
						knownvalue.Bool(true),
					),
				},
			},
		},
	})
}

func policyConfigDockItems(policyName, dockItemName, action string) string {
	return fmt.Sprintf(`
resource "jamfplatform_pro_dock_item" "fixture" {
  name = %q
  type = "App"
  path = "/Applications/Calculator.app"
}

resource "jamfplatform_pro_policy" "test" {
  general = {
    name = %q
  }
  dock_items = {
    dock_items = [
      {
        id     = jamfplatform_pro_dock_item.fixture.id
        action = %q
      },
    ]
  }
}
`, dockItemName, policyName, action)
}

// TestAccPolicyResource_DockItemsFullCoverage creates a jamfplatform_pro_dock_item
// fixture and references it from policy.dock_items.dock_items. Step 2 swaps action
// `Add To End` → `Add To Beginning` to exercise the Update path.
func TestAccPolicyResource_DockItemsFullCoverage(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	policyName := "tf-acc-policy-dockitems-" + suffix
	dockItemName := "tf-acc-dock-item-fixture-" + suffix

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPolicyDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: policyConfigDockItems(policyName, dockItemName, "Add To End"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("dock_items").AtMapKey("dock_items"),
						knownvalue.SetSizeExact(1),
					),
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("dock_items").AtMapKey("dock_items").AtSliceIndex(0).AtMapKey("action"),
						knownvalue.StringExact("Add To End"),
					),
				},
			},
			{
				Config: policyConfigDockItems(policyName, dockItemName, "Add To Beginning"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("dock_items").AtMapKey("dock_items").AtSliceIndex(0).AtMapKey("action"),
						knownvalue.StringExact("Add To Beginning"),
					),
				},
			},
		},
	})
}

func policyConfigMaintenance(name string, recon bool) string {
	return fmt.Sprintf(`
resource "jamfplatform_pro_policy" "test" {
  general = {
    name = %q
  }
  maintenance = {
    recon                       = %t
    reset_name                  = true
    install_all_cached_packages = true
    heal                        = false
    prebindings                 = false
    permissions                 = true
    byhost                      = true
    system_cache                = true
    user_cache                  = true
    verify                      = true
  }
}
`, name, recon)
}

// TestAccPolicyResource_MaintenanceFullCoverage exercises every maintenance
// attribute. Per Probe #10 in PHASE_2_6_SPIKE.md, the wire silently rejects
// heal=true (echo always returns false) so this test sets heal=false to match
// what the server will return; the schema still surfaces the attribute for
// users who want to declare the inert wire field. Step 2 toggles recon to
// exercise the Update path.
func TestAccPolicyResource_MaintenanceFullCoverage(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-policy-maint-" + suffix

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPolicyDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: policyConfigMaintenance(name, true),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("maintenance").AtMapKey("recon"),
						knownvalue.Bool(true),
					),
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("maintenance").AtMapKey("heal"),
						knownvalue.Bool(false),
					),
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("maintenance").AtMapKey("verify"),
						knownvalue.Bool(true),
					),
				},
			},
			{
				Config: policyConfigMaintenance(name, false),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("maintenance").AtMapKey("recon"),
						knownvalue.Bool(false),
					),
				},
			},
		},
	})
}

func policyConfigFilesProcesses(name, searchByPath, runCommand string) string {
	return fmt.Sprintf(`
resource "jamfplatform_pro_policy" "test" {
  general = {
    name = %q
  }
  files_processes = {
    search_by_path         = %q
    delete_file            = true
    locate_file            = "tf-acc-locate-file"
    update_locate_database = true
    spotlight_search       = "tf-acc-spotlight"
    search_for_process     = "tf-acc-process"
    kill_process           = true
    run_command            = %q
  }
}
`, name, searchByPath, runCommand)
}

// TestAccPolicyResource_FilesProcessesFullCoverage exercises every
// files_processes attribute. Step 2 perturbs two scalar string fields to
// exercise the Update path.
func TestAccPolicyResource_FilesProcessesFullCoverage(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-policy-filesproc-" + suffix

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPolicyDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: policyConfigFilesProcesses(name, "/tmp/tf-acc-search", "echo tf-acc-initial"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("files_processes").AtMapKey("search_by_path"),
						knownvalue.StringExact("/tmp/tf-acc-search"),
					),
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("files_processes").AtMapKey("delete_file"),
						knownvalue.Bool(true),
					),
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("files_processes").AtMapKey("locate_file"),
						knownvalue.StringExact("tf-acc-locate-file"),
					),
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("files_processes").AtMapKey("kill_process"),
						knownvalue.Bool(true),
					),
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("files_processes").AtMapKey("run_command"),
						knownvalue.StringExact("echo tf-acc-initial"),
					),
				},
			},
			{
				Config: policyConfigFilesProcesses(name, "/tmp/tf-acc-search-updated", "echo tf-acc-updated"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("files_processes").AtMapKey("search_by_path"),
						knownvalue.StringExact("/tmp/tf-acc-search-updated"),
					),
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("files_processes").AtMapKey("run_command"),
						knownvalue.StringExact("echo tf-acc-updated"),
					),
				},
			},
		},
	})
}

func policyConfigUserInteractionUntilUTC(name, messageStart, untilUTC string) string {
	return fmt.Sprintf(`
resource "jamfplatform_pro_policy" "test" {
  general = {
    name = %q
  }
  user_interaction = {
    message_start            = %q
    allow_users_to_defer     = true
    allow_deferral_until_utc = %q
    message_finish           = "tf-acc message_finish"
  }
}
`, name, messageStart, untilUTC)
}

// TestAccPolicyResource_UserInteractionFullCoverage exercises every
// user_interaction attribute. Both steps use the cut-off form
// (allow_deferral_until_utc); step 2 perturbs message_start and the cut-off
// date to exercise the Update path. The duration form
// (allow_deferral_minutes) is documented but not toggled in the same test
// because the classic API cannot transition between the two forms without an
// intermediate clear (see constraint notes below) and step coverage of the
// minutes form belongs in a separate, dedicated test.
//
// Three server-side cross-field constraints surfaced during this test and
// are noted here for follow-up plan-time validators:
//   - allow_deferral_minutes must be a multiple of 1440 (one day): the server
//     returns 409 with `Error: allow_deferral_minutes must be a multiple of
//     1440 (minutes in day), currently the UI displays only number of days`.
//   - When allow_users_to_defer=false, allow_deferral_until_utc and
//     allow_deferral_minutes cannot be configured: the server returns 409
//     with `Error: When 'allow_users_to_defer' is false,
//     'allow_deferral_until_utc' and 'allow_deferral_minutes' cannot be
//     configured`.
//   - allow_deferral_until_utc and allow_deferral_minutes are mutually
//     exclusive: the server returns 409 with `Error: You cannot use both
//     'allow_deferral_until_utc' and 'allow_deferral_minutes'`. Omitting one
//     field on an Update does not clear it (Optional+Computed semantics keep
//     prior state); transitioning between forms requires destroy+recreate.
func TestAccPolicyResource_UserInteractionFullCoverage(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-policy-userint-" + suffix

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPolicyDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: policyConfigUserInteractionUntilUTC(name, "tf-acc message_start initial", "2030-01-01T01:00:00.000+0000"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("user_interaction").AtMapKey("message_start"),
						knownvalue.StringExact("tf-acc message_start initial"),
					),
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("user_interaction").AtMapKey("allow_users_to_defer"),
						knownvalue.Bool(true),
					),
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("user_interaction").AtMapKey("allow_deferral_until_utc"),
						knownvalue.StringExact("2030-01-01T01:00:00.000+0000"),
					),
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("user_interaction").AtMapKey("message_finish"),
						knownvalue.StringExact("tf-acc message_finish"),
					),
				},
			},
			{
				Config: policyConfigUserInteractionUntilUTC(name, "tf-acc message_start updated", "2031-06-15T12:00:00.000+0000"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("user_interaction").AtMapKey("message_start"),
						knownvalue.StringExact("tf-acc message_start updated"),
					),
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("user_interaction").AtMapKey("allow_users_to_defer"),
						knownvalue.Bool(true),
					),
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("user_interaction").AtMapKey("allow_deferral_until_utc"),
						knownvalue.StringExact("2031-06-15T12:00:00.000+0000"),
					),
				},
			},
		},
	})
}

func policyConfigDiskEncryption(policyName, decName, action string, authRestart bool) string {
	return fmt.Sprintf(`
resource "jamfplatform_pro_disk_encryption_configuration" "fixture" {
  name                     = %q
  key_type                 = "Individual"
  file_vault_enabled_users = "Current or Next User"
}

resource "jamfplatform_pro_policy" "test" {
  general = {
    name = %q
  }
  disk_encryption = {
    action                           = %q
    disk_encryption_configuration_id = tonumber(jamfplatform_pro_disk_encryption_configuration.fixture.id)
    auth_restart                     = %t
  }
}
`, decName, policyName, action, authRestart)
}

// TestAccPolicyResource_DiskEncryptionFullCoverage creates a
// jamfplatform_pro_disk_encryption_configuration fixture (key_type=Individual
// is the minimal config — no IRK certificate required) and references its ID
// from policy.disk_encryption. Step 2 toggles auth_restart to exercise the
// Update path. Probe #12 in PHASE_2_6_SPIKE.md found that action=remediate
// silently reverts to apply on the server, so this test sticks with
// action=apply throughout.
func TestAccPolicyResource_DiskEncryptionFullCoverage(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	policyName := "tf-acc-policy-de-" + suffix
	decName := "tf-acc-dec-fixture-" + suffix

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPolicyDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: policyConfigDiskEncryption(policyName, decName, "apply", false),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("disk_encryption").AtMapKey("action"),
						knownvalue.StringExact("apply"),
					),
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("disk_encryption").AtMapKey("auth_restart"),
						knownvalue.Bool(false),
					),
				},
			},
			{
				Config: policyConfigDiskEncryption(policyName, decName, "apply", true),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("disk_encryption").AtMapKey("auth_restart"),
						knownvalue.Bool(true),
					),
				},
			},
		},
	})
}
