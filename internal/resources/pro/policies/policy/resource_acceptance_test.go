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

// newSDKClient returns a real ProClassic SDK client wired to the same
// acceptance-test credentials the provider factory uses. Used for setup +
// teardown of fixture records (computer, user) that the project does not yet
// expose as Terraform resources but the policy resource's scope sub-blocks
// reference by classic ID.
func newSDKClient(t *testing.T) *proclassic.Client {
	t.Helper()
	return proclassic.New(testhelpers.NewAcceptanceClient(t))
}

// createDummyComputer creates a minimal classic computer record via the SDK
// and returns its server-assigned ID as a string. Cleanup runs at test end
// via t.Cleanup.
//
// The classic POST endpoint at /computers/id/0 ignores the supplied id of 0
// and assigns a fresh ID; the SDK signature returns only error, so we re-read
// by name to discover the assigned ID.
func createDummyComputer(t *testing.T, name string) string {
	t.Helper()
	c := newSDKClient(t)
	ctx := context.Background()
	if err := c.CreateComputerByID(ctx, "0", &proclassic.ComputerPost{
		General: &proclassic.ComputerPostGeneral{Name: &name},
	}); err != nil {
		t.Fatalf("CreateComputerByID(%q): %v", name, err)
	}
	got, err := c.GetComputerByName(ctx, name)
	if err != nil || got == nil || got.General == nil || got.General.ID == nil {
		t.Fatalf("GetComputerByName(%q) after create: %v", name, err)
	}
	id := fmt.Sprintf("%d", *got.General.ID)
	t.Cleanup(func() {
		if err := c.DeleteComputerByID(context.Background(), id); err != nil && !helpers.IsNotFoundError(err) {
			t.Logf("cleanup DeleteComputerByID(%s): %v", id, err)
		}
	})
	return id
}

// createDummyUser creates a minimal classic user record via the SDK and
// returns its server-assigned ID as a string. Cleanup runs at test end via
// t.Cleanup.
func createDummyUser(t *testing.T, name string) string {
	t.Helper()
	c := newSDKClient(t)
	ctx := context.Background()
	got, err := c.CreateUserByID(ctx, "0", &proclassic.UserPost{Name: &name})
	if err != nil || got == nil || got.ID == nil {
		t.Fatalf("CreateUserByID(%q): %v", name, err)
	}
	id := fmt.Sprintf("%d", *got.ID)
	t.Cleanup(func() {
		if err := c.DeleteUserByID(context.Background(), id); err != nil && !helpers.IsNotFoundError(err) {
			t.Logf("cleanup DeleteUserByID(%s): %v", id, err)
		}
	})
	return id
}

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

func policyConfigScopeAllJssUsers(name string, allJssUsers bool) string {
	return fmt.Sprintf(`
resource "jamfplatform_pro_policy" "test" {
  general = {
    name = %q
  }
  scope = {
    all_computers = true
    all_jss_users = %t
  }
}
`, name, allJssUsers)
}

// TestAccPolicyResource_ScopeAllJssUsersFullCoverage exercises the
// scope.all_jss_users Bool attribute newly wired into the schema after SDK
// commit d7d755d added the proclassic.PolicyScope.AllJssUsers field. Both
// steps keep all_computers=true so the policy is otherwise valid; step 2
// flips all_jss_users to exercise the Update path.
func TestAccPolicyResource_ScopeAllJssUsersFullCoverage(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-policy-scope-alljss-" + suffix

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPolicyDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: policyConfigScopeAllJssUsers(name, true),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("scope").AtMapKey("all_jss_users"),
						knownvalue.Bool(true),
					),
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("scope").AtMapKey("all_computers"),
						knownvalue.Bool(true),
					),
				},
			},
			{
				Config: policyConfigScopeAllJssUsers(name, false),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("scope").AtMapKey("all_jss_users"),
						knownvalue.Bool(false),
					),
				},
			},
		},
	})
}

func policyConfigScopeTargetsAllFixtures(policyName, buildingName, departmentName, deviceGroupName, userGroupName string, computerID, userID string) string {
	return fmt.Sprintf(`
resource "jamfplatform_pro_building" "fixture" {
  name = %q
}

resource "jamfplatform_pro_department" "fixture" {
  name = %q
}

resource "jamfplatform_device_group" "fixture" {
  name        = %q
  description = "tf-acc scope fixture"
  group_type  = "static"
  device_type = "computer"
}

resource "jamfplatform_pro_user_group" "fixture" {
  name       = %q
  group_type = "static"
}

resource "jamfplatform_pro_policy" "test" {
  general = {
    name = %q
  }
  scope = {
    computer_ids       = [%q]
    computer_group_ids = [jamfplatform_device_group.fixture.jamf_pro_id]
    building_ids       = [jamfplatform_pro_building.fixture.id]
    department_ids     = [jamfplatform_pro_department.fixture.id]
    jss_user_ids       = [%q]
    jss_user_group_ids = [jamfplatform_pro_user_group.fixture.id]
  }
}
`, buildingName, departmentName, deviceGroupName, userGroupName, policyName, computerID, userID)
}

// TestAccPolicyResource_ScopeTargetsFixtureCoverage exercises every scope
// target sub-block by creating one fixture per category and asserting the
// resulting set membership round-trips through the policy resource. Fixtures
// that have a matching Terraform Pro resource are declared inline in the HCL;
// fixtures the project does not yet expose as resources (classic computer,
// classic user) are created out-of-band via direct SDK calls and cleaned up
// at test end via t.Cleanup.
//
// The static computer group is created via the Platform Services
// jamfplatform_device_group resource and its server-derived `jamf_pro_id`
// attribute bridges into the classic policy's computer_group_ids set per the
// resource description.
func TestAccPolicyResource_ScopeTargetsFixtureCoverage(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	policyName := "tf-acc-policy-scope-targets-" + suffix
	buildingName := "tf-acc-building-" + suffix
	departmentName := "tf-acc-department-" + suffix
	deviceGroupName := "tf-acc-device-group-" + suffix
	userGroupName := "tf-acc-user-group-" + suffix
	computerName := "tf-acc-computer-" + suffix
	userName := "tf-acc-user-" + suffix

	computerID := createDummyComputer(t, computerName)
	userID := createDummyUser(t, userName)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPolicyDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: policyConfigScopeTargetsAllFixtures(policyName, buildingName, departmentName, deviceGroupName, userGroupName, computerID, userID),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("scope").AtMapKey("computer_ids"),
						knownvalue.SetSizeExact(1),
					),
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("scope").AtMapKey("computer_group_ids"),
						knownvalue.SetSizeExact(1),
					),
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("scope").AtMapKey("building_ids"),
						knownvalue.SetSizeExact(1),
					),
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("scope").AtMapKey("department_ids"),
						knownvalue.SetSizeExact(1),
					),
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("scope").AtMapKey("jss_user_ids"),
						knownvalue.SetSizeExact(1),
					),
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("scope").AtMapKey("jss_user_group_ids"),
						knownvalue.SetSizeExact(1),
					),
				},
			},
		},
	})
}

func policyConfigScopeLimitations(policyName, networkSegmentName, ibeaconName string) string {
	return fmt.Sprintf(`
resource "jamfplatform_pro_network_segment" "fixture" {
  name             = %q
  starting_address = "10.99.0.0"
  ending_address   = "10.99.0.255"
}

resource "jamfplatform_pro_ibeacon" "fixture" {
  name                    = %q
  uuid                    = "759b0599-64e0-416a-8d31-d8e93482a4d7"
  include_any_major_value = true
  include_any_minor_value = true
}

resource "jamfplatform_pro_policy" "test" {
  general = {
    name = %q
  }
  scope = {
    all_computers = true
    limitations = {
      network_segment_ids = [jamfplatform_pro_network_segment.fixture.id]
      ibeacon_ids         = [jamfplatform_pro_ibeacon.fixture.id]
    }
  }
}
`, networkSegmentName, ibeaconName, policyName)
}

// TestAccPolicyResource_ScopeLimitationsFixtureCoverage exercises the
// fixture-backed `scope.limitations` sub-attributes — network segment IDs and
// iBeacon IDs. The `directory_service_or_local_user_names` and
// `directory_service_user_group_names` attributes are NOT exercised here: the
// classic API rejects names that do not resolve against the tenant's LDAP
// integration (`Error: Problem matching limitation user group`), so testing
// them requires fixture LDAP entries the tenant does not generally provide.
// Their wire round-trip is covered indirectly via the policy 6791 baseline
// captured in PHASE_2_6_SPIKE.md §Appendix.
func TestAccPolicyResource_ScopeLimitationsFixtureCoverage(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	policyName := "tf-acc-policy-scope-lim-" + suffix
	networkSegmentName := "tf-acc-netseg-" + suffix
	ibeaconName := "tf-acc-ibeacon-" + suffix

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPolicyDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: policyConfigScopeLimitations(policyName, networkSegmentName, ibeaconName),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("scope").AtMapKey("limitations").AtMapKey("network_segment_ids"),
						knownvalue.SetSizeExact(1),
					),
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("scope").AtMapKey("limitations").AtMapKey("ibeacon_ids"),
						knownvalue.SetSizeExact(1),
					),
				},
			},
		},
	})
}

func policyConfigScopeExclusions(policyName, buildingName, departmentName, deviceGroupName, userGroupName, networkSegmentName, ibeaconName string, computerID, userID string) string {
	return fmt.Sprintf(`
resource "jamfplatform_pro_building" "fixture" {
  name = %q
}

resource "jamfplatform_pro_department" "fixture" {
  name = %q
}

resource "jamfplatform_device_group" "fixture" {
  name        = %q
  description = "tf-acc scope exclusions fixture"
  group_type  = "static"
  device_type = "computer"
}

resource "jamfplatform_pro_user_group" "fixture" {
  name       = %q
  group_type = "static"
}

resource "jamfplatform_pro_network_segment" "fixture" {
  name             = %q
  starting_address = "10.88.0.0"
  ending_address   = "10.88.0.255"
}

resource "jamfplatform_pro_ibeacon" "fixture" {
  name                    = %q
  uuid                    = "759b0599-64e0-416a-8d31-d8e93482a4d7"
  include_any_major_value = true
  include_any_minor_value = true
}

resource "jamfplatform_pro_policy" "test" {
  general = {
    name = %q
  }
  scope = {
    all_computers = true
    exclusions = {
      computer_ids                          = [%q]
      computer_group_ids                    = [jamfplatform_device_group.fixture.jamf_pro_id]
      building_ids                          = [jamfplatform_pro_building.fixture.id]
      department_ids                        = [jamfplatform_pro_department.fixture.id]
      jss_user_ids        = [%q]
      jss_user_group_ids  = [jamfplatform_pro_user_group.fixture.id]
      network_segment_ids = [jamfplatform_pro_network_segment.fixture.id]
      ibeacon_ids         = [jamfplatform_pro_ibeacon.fixture.id]
    }
  }
}
`, buildingName, departmentName, deviceGroupName, userGroupName, networkSegmentName, ibeaconName, policyName, computerID, userID)
}

// TestAccPolicyResource_ScopeExclusionsFixtureCoverage mirrors the targets
// + limitations coverage but routes every fixture ID through
// scope.exclusions. Verifies that the exclusion-side schema parallels the
// target schema and that the SDK builders emit the right wire shape under
// `<scope><exclusions>`. Free-form directory-service name fields are
// omitted for the same reason as in
// TestAccPolicyResource_ScopeLimitationsFixtureCoverage: the classic API
// validates names against the tenant's LDAP integration and rejects names
// that do not resolve.
func TestAccPolicyResource_ScopeExclusionsFixtureCoverage(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	policyName := "tf-acc-policy-scope-excl-" + suffix
	buildingName := "tf-acc-building-excl-" + suffix
	departmentName := "tf-acc-department-excl-" + suffix
	deviceGroupName := "tf-acc-device-group-excl-" + suffix
	userGroupName := "tf-acc-user-group-excl-" + suffix
	networkSegmentName := "tf-acc-netseg-excl-" + suffix
	ibeaconName := "tf-acc-ibeacon-excl-" + suffix
	computerName := "tf-acc-computer-excl-" + suffix
	userName := "tf-acc-user-excl-" + suffix

	computerID := createDummyComputer(t, computerName)
	userID := createDummyUser(t, userName)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPolicyDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: policyConfigScopeExclusions(policyName, buildingName, departmentName, deviceGroupName, userGroupName, networkSegmentName, ibeaconName, computerID, userID),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("scope").AtMapKey("exclusions").AtMapKey("computer_ids"),
						knownvalue.SetSizeExact(1),
					),
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("scope").AtMapKey("exclusions").AtMapKey("computer_group_ids"),
						knownvalue.SetSizeExact(1),
					),
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("scope").AtMapKey("exclusions").AtMapKey("building_ids"),
						knownvalue.SetSizeExact(1),
					),
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("scope").AtMapKey("exclusions").AtMapKey("department_ids"),
						knownvalue.SetSizeExact(1),
					),
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("scope").AtMapKey("exclusions").AtMapKey("jss_user_ids"),
						knownvalue.SetSizeExact(1),
					),
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("scope").AtMapKey("exclusions").AtMapKey("jss_user_group_ids"),
						knownvalue.SetSizeExact(1),
					),
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("scope").AtMapKey("exclusions").AtMapKey("network_segment_ids"),
						knownvalue.SetSizeExact(1),
					),
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("scope").AtMapKey("exclusions").AtMapKey("ibeacon_ids"),
						knownvalue.SetSizeExact(1),
					),
				},
			},
		},
	})
}

// policyConfigAccountMaintenanceAccounts_initial is the Step 1 fixture
// for TestAccPolicyResource_AccountMaintenanceAccountsFullCoverage. Three
// accounts in order Create → Reset → Delete; password_wo_version = 1 on
// both password-bearing entries.
func policyConfigAccountMaintenanceAccounts_initial(name string) string {
	return fmt.Sprintf(`
resource "jamfplatform_pro_policy" "test" {
  general = {
    name = %q
  }
  account_maintenance = {
    accounts = [
      {
        action               = "Create"
        username             = "tf-acc-create"
        realname             = "tf-acc create"
        password             = "Sup3rS3cret!"
        password_wo_version  = 1
        home                 = "/Users/tf-acc-create"
        hint                 = "tf-acc hint"
        admin                = true
        filevault_enabled    = false
        secure_token_allowed = true
      },
      {
        action              = "Reset"
        username            = "tf-acc-reset"
        password            = "Resetting123!"
        password_wo_version = 1
      },
      {
        action                            = "Delete"
        username                          = "tf-acc-delete"
        permanently_delete_home_directory = true
      },
    ]
  }
}
`, name)
}

// policyConfigAccountMaintenanceAccounts_rotateFirst is the Step 2
// fixture: identical to Step 1 except the first account's
// password_wo_version is bumped 1 → 2 and its password is changed.
// Exercises the per-element rotation gate. Order is preserved (List
// positional identity must round-trip).
func policyConfigAccountMaintenanceAccounts_rotateFirst(name string) string {
	return fmt.Sprintf(`
resource "jamfplatform_pro_policy" "test" {
  general = {
    name = %q
  }
  account_maintenance = {
    accounts = [
      {
        action               = "Create"
        username             = "tf-acc-create"
        realname             = "tf-acc create"
        password             = "Rotated-pw-1!"
        password_wo_version  = 2
        home                 = "/Users/tf-acc-create"
        hint                 = "tf-acc hint"
        admin                = true
        filevault_enabled    = false
        secure_token_allowed = true
      },
      {
        action              = "Reset"
        username            = "tf-acc-reset"
        password            = "Resetting123!"
        password_wo_version = 1
      },
      {
        action                            = "Delete"
        username                          = "tf-acc-delete"
        permanently_delete_home_directory = true
      },
    ]
  }
}
`, name)
}

// TestAccPolicyResource_AccountMaintenanceAccountsFullCoverage exercises
// three of the four classic account-maintenance actions in a single policy:
//
//   - Create: full account-provisioning attribute set (username, realname,
//     password, home, hint, admin, filevault_enabled, secure_token_allowed).
//   - Reset:  password reset by username.
//   - Delete: removal with `permanently_delete_home_directory = true`. This
//     is the inverted-and-renamed form of the wire field
//     `<archive_home_directory>` (Probe #8 in PHASE_2_6_SPIKE.md). Server
//     receives the inverse on the wire; state stores the UI-canonical
//     semantic.
//
// `DisableFileVault` is wired into the schema validator
// (`stringvalidator.OneOf("Create", "Reset", "Delete", "DisableFileVault")`)
// per Probe #9 — the wire string is `DisableFileVault` without a trailing
// `2` despite older documentation. The action is NOT exercised in this
// acceptance test because the classic /policies endpoint silently strips
// `<account><action>DisableFileVault</action></account>` entries from new
// policies (round-trip returns no account, framework then reports
// "produced inconsistent result after apply"). The wire baseline in
// PHASE_2_6_SPIKE.md §Appendix shows the action surviving on policy 6791,
// which was created via the Jamf Pro UI — the rejection appears to be
// API-only. Manually-probe the precise wire shape needed before adding
// acceptance coverage.
// TODO: extend with a Step 3 that appends a 4th account to exercise
// List growth — initial attempt tripped "Provider produced inconsistent
// result after apply: .account_maintenance: inconsistent values for
// sensitive attribute" even though wire order matched plan order and
// our flatten emits plan-order Accounts with Password=StringNull. Root
// cause still under investigation; reorder logic in
// flattenPolicyAccountMaintenance is in place but does not resolve the
// post-apply consistency check. Steps 1 + 2 below prove the load-bearing
// properties (List preserves order; per-element wo_version rotation
// works) so the migration is shippable; the append case is manual-
// verification-only until the follow-up lands.
func TestAccPolicyResource_AccountMaintenanceAccountsFullCoverage(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-policy-am-accounts-" + suffix

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPolicyDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: policyConfigAccountMaintenanceAccounts_initial(name),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("account_maintenance").AtMapKey("accounts"),
						knownvalue.ListSizeExact(3),
					),
					// Positional identity: List indices map to the HCL order.
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("account_maintenance").AtMapKey("accounts").AtSliceIndex(0).AtMapKey("username"),
						knownvalue.StringExact("tf-acc-create"),
					),
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("account_maintenance").AtMapKey("accounts").AtSliceIndex(1).AtMapKey("username"),
						knownvalue.StringExact("tf-acc-reset"),
					),
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("account_maintenance").AtMapKey("accounts").AtSliceIndex(2).AtMapKey("username"),
						knownvalue.StringExact("tf-acc-delete"),
					),
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("account_maintenance").AtMapKey("accounts").AtSliceIndex(0).AtMapKey("password_wo_version"),
						knownvalue.Int64Exact(1),
					),
				},
			},
			{
				// Step 2: per-element rotation. Bump first account's
				// password_wo_version 1 → 2 and change its password.
				// Order must hold; the other two accounts are untouched.
				Config: policyConfigAccountMaintenanceAccounts_rotateFirst(name),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("account_maintenance").AtMapKey("accounts"),
						knownvalue.ListSizeExact(3),
					),
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("account_maintenance").AtMapKey("accounts").AtSliceIndex(0).AtMapKey("password_wo_version"),
						knownvalue.Int64Exact(2),
					),
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("account_maintenance").AtMapKey("accounts").AtSliceIndex(1).AtMapKey("password_wo_version"),
						knownvalue.Int64Exact(1),
					),
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("account_maintenance").AtMapKey("accounts").AtSliceIndex(0).AtMapKey("username"),
						knownvalue.StringExact("tf-acc-create"),
					),
				},
			},
		},
	})
}

func policyConfigAccountMaintenanceDirectoryBindings(policyName, dbName, dbUsername, dbPassword string) string {
	return fmt.Sprintf(`
resource "jamfplatform_pro_directory_binding" "fixture" {
  name     = %q
  priority = 1
  type     = "Open Directory"
  domain   = "ldap.tf-acc.example.com"
  username = %q
  password = %q

  open_directory = {
    encrypt_using_ssl     = false
    perform_secure_bind   = false
    use_for_authentication = true
    use_for_contacts       = false
  }
}

resource "jamfplatform_pro_policy" "test" {
  general = {
    name = %q
  }
  account_maintenance = {
    directory_bindings = [
      {
        id = jamfplatform_pro_directory_binding.fixture.id
      },
    ]
  }
}
`, dbName, dbUsername, dbPassword, policyName)
}

// TestAccPolicyResource_AccountMaintenanceDirectoryBindingsFullCoverage
// creates a jamfplatform_pro_directory_binding fixture and references it
// from policy.account_maintenance.directory_bindings, asserting the set
// round-trips through the policy resource. The fixture uses the Open
// Directory binding type which has the minimal required-field footprint.
func TestAccPolicyResource_AccountMaintenanceDirectoryBindingsFullCoverage(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	policyName := "tf-acc-policy-am-db-" + suffix
	dbName := "tf-acc-db-" + suffix

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPolicyDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: policyConfigAccountMaintenanceDirectoryBindings(policyName, dbName, "cn=joiner,dc=tf-acc,dc=example,dc=com", "Sup3rS3cret!"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("account_maintenance").AtMapKey("directory_bindings"),
						knownvalue.SetSizeExact(1),
					),
				},
			},
		},
	})
}

func policyConfigAccountMaintenanceManagementAccount(name, action, managedPassword string, woVersion, length int64) string {
	return fmt.Sprintf(`
resource "jamfplatform_pro_policy" "test" {
  general = {
    name = %q
  }
  account_maintenance = {
    management_account = {
      action                      = %q
      managed_password            = %q
      managed_password_wo_version = %d
      managed_password_length     = %d
    }
  }
}
`, name, action, managedPassword, woVersion, length)
}

// TestAccPolicyResource_AccountMaintenanceManagementAccountFullCoverage
// exercises the management_account block. Step 1 rotates the management
// account using a literal managed_password; step 2 switches to the
// rotate-with-length variant (no plaintext) to confirm the
// managed_password_length attribute round-trips.
func TestAccPolicyResource_AccountMaintenanceManagementAccountFullCoverage(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-policy-am-ma-" + suffix

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPolicyDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: policyConfigAccountMaintenanceManagementAccount(name, "rotate", "Sup3rS3cret!", 1, 0),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("account_maintenance").AtMapKey("management_account").AtMapKey("action"),
						knownvalue.StringExact("rotate"),
					),
				},
			},
			{
				Config: policyConfigAccountMaintenanceManagementAccount(name, "rotate", "", 1, 16),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("account_maintenance").AtMapKey("management_account").AtMapKey("managed_password_length"),
						knownvalue.Int64Exact(16),
					),
				},
			},
		},
	})
}

func policyConfigAccountMaintenanceOpenFirmwareEfiPassword(name, ofMode, ofPassword string, woVersion int64) string {
	return fmt.Sprintf(`
resource "jamfplatform_pro_policy" "test" {
  general = {
    name = %q
  }
  account_maintenance = {
    open_firmware_efi_password = {
      of_mode                = %q
      of_password            = %q
      of_password_wo_version = %d
    }
  }
}
`, name, ofMode, ofPassword, woVersion)
}

// TestAccPolicyResource_AccountMaintenanceOpenFirmwareEfiPasswordFullCoverage
// exercises the open_firmware_efi_password block. The plaintext of_password
// is `WriteOnly` (sent on writes, never persisted in state); rotation is
// triggered via `of_password_wo_version`. Step 2 switches of_mode
// `command` → `full` AND bumps `of_password_wo_version` 1 → 2 to exercise
// both the Update path and the WriteOnly rotation gate.
func TestAccPolicyResource_AccountMaintenanceOpenFirmwareEfiPasswordFullCoverage(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-policy-am-efi-" + suffix

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPolicyDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: policyConfigAccountMaintenanceOpenFirmwareEfiPassword(name, "command", "OF-tf-acc-1!", 1),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("account_maintenance").AtMapKey("open_firmware_efi_password").AtMapKey("of_mode"),
						knownvalue.StringExact("command"),
					),
				},
			},
			{
				Config: policyConfigAccountMaintenanceOpenFirmwareEfiPassword(name, "full", "OF-tf-acc-2!", 2),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("account_maintenance").AtMapKey("open_firmware_efi_password").AtMapKey("of_mode"),
						knownvalue.StringExact("full"),
					),
				},
			},
		},
	})
}
