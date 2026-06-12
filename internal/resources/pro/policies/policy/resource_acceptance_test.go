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
	"regexp"
	"runtime"
	"strings"
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
    display_notifications      = true
    notification_location       = "Self Service"
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
    targets = {
      all_computers = true
    }
  }
}
`, name)
}

// TestAccPolicyResource_ScopeTargetsNullToPresentTransition is the load-bearing
// regression test for the `targets` nesting: it exercises why the all-flags use
// boolplanmodifier.UseNonNullStateForUnknown rather than UseStateForUnknown.
//
// Step 1 declares `scope` with only `exclusions`, so the `targets` block is
// absent and the Computed all-flags have a NULL prior state. Step 2 adds
// `targets { all_jss_users = true }` while leaving `all_computers` Computed
// (omitted from config) — so `all_computers` undergoes the null→present block
// transition as an unknown-at-plan value. Under the old UseStateForUnknown the
// modifier would carry the null prior state into the plan and trip a
// "produced an inconsistent result after apply … was null, but now <bool>"
// error once the server echoes a concrete value; UseNonNullStateForUnknown
// leaves it unknown so apply fills it cleanly. A green step 2 IS the assertion.
func TestAccPolicyResource_ScopeTargetsNullToPresentTransition(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-policy-targets-transition-" + suffix

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPolicyDestroy(t),
		Steps: []resource.TestStep{
			{
				// scope present, targets ABSENT → null prior state.
				Config: policyConfigScopeExclusionsOnly(name),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("scope").AtMapKey("targets"),
						knownvalue.Null(),
					),
				},
			},
			{
				// targets goes null→present; all_computers stays Computed.
				Config: policyConfigScopeExclusionsPlusTargets(name),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("scope").AtMapKey("targets").AtMapKey("all_jss_users"),
						knownvalue.Bool(true),
					),
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("scope").AtMapKey("targets").AtMapKey("all_computers"),
						knownvalue.Bool(false),
					),
				},
			},
		},
	})
}

func policyConfigScopeExclusionsOnly(name string) string {
	return fmt.Sprintf(`
resource "jamfplatform_pro_policy" "test" {
  general = {
    name = %q
  }
  scope = {
    exclusions = {
      directory_service_or_local_user_names = ["tf-acc-excluded-user"]
    }
  }
}
`, name)
}

func policyConfigScopeExclusionsPlusTargets(name string) string {
	return fmt.Sprintf(`
resource "jamfplatform_pro_policy" "test" {
  general = {
    name = %q
  }
  scope = {
    targets = {
      all_jss_users = true
    }
    exclusions = {
      directory_service_or_local_user_names = ["tf-acc-excluded-user"]
    }
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
// display_notifications + notification_location attributes.
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
						tfjsonpath.New("self_service").AtMapKey("display_notifications"),
						knownvalue.Bool(true),
					),
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("self_service").AtMapKey("notification_location"),
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
						tfjsonpath.New("scope").AtMapKey("targets").AtMapKey("all_computers"),
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
    trigger_network_state_changed = true
    trigger_startup               = true
    trigger_other                 = "tf-acc-event"
    frequency                     = "Once per computer"
    retry_event                   = "check-in"
    retry_attempts                = 3
    notify_on_each_failed_retry   = true
    limit_to_jamf_pro_assigned_user            = false
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
						tfjsonpath.New("general").AtMapKey("date_time_limitations").AtMapKey("no_execute_end"),
						knownvalue.StringExact("2:00 AM"),
					),
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("general").AtMapKey("date_time_limitations").AtMapKey("no_execute_on"),
						knownvalue.SetExact([]knownvalue.Check{
							knownvalue.StringExact("Sat"),
							knownvalue.StringExact("Sun"),
						}),
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
    delay_minutes           = 10
    start_reboot_timer_immediately = true
    file_vault_2_reboot            = false
  }
}
`, name, specifyStartup)
}

// TestAccPolicyResource_RebootFullCoverage exercises every reboot attribute
// and sweeps specify_startup through all three values the validator accepts:
// the wire-empty default, the standard restart label, and the MDM
// kernel-cache-rebuild label.
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
// SDK 0.8.1-…46aec40edb28. The wire returns this value as a peer of <packages>.
// Test omits the `packages` set entirely — there is no clean "NONE" value for
// packages[].id, so per-package
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
						tfjsonpath.New("reboot").AtMapKey("delay_minutes"),
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
// `priority` is `Before`/`After`/`At Reboot`, distinct from the fixture's
// enum. Step 2 swaps priority and
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
// `Map` → `Unmap` (the UI-canonical values exposed by the schema; the provider
// translates to/from the wire `install`/`uninstall` form via printerActionToWire
// / printerActionFromWire) and toggles make_default + leave_existing_default
// to exercise the Update path.
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
				Config: policyConfigPrinters(policyName, printerName, "Map", true, false),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("printers").AtMapKey("printers"),
						knownvalue.SetSizeExact(1),
					),
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("printers").AtMapKey("printers").AtSliceIndex(0).AtMapKey("action"),
						knownvalue.StringExact("Map"),
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
				Config: policyConfigPrinters(policyName, printerName, "Unmap", false, true),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("printers").AtMapKey("printers").AtSliceIndex(0).AtMapKey("action"),
						knownvalue.StringExact("Unmap"),
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

func policyConfigMaintenance(name string, updateInventory bool) string {
	return fmt.Sprintf(`
resource "jamfplatform_pro_policy" "test" {
  general = {
    name = %q
  }
  maintenance = {
    update_inventory        = %t
    reset_computer_names    = true
    install_cached_packages = true
    fix_disk_permissions    = true
    fix_byhost_files        = true
    flush_system_caches     = true
    flush_user_caches       = true
    verify_startup_disk     = true
  }
}
`, name, updateInventory)
}

// TestAccPolicyResource_MaintenanceFullCoverage exercises every maintenance
// attribute. Step 2 toggles update_inventory to exercise the Update path.
// The wire-dead `heal` and `prebindings` fields (wire rejected or echoed
// empty regardless of the value sent) were dropped from the schema in the
// same PR as this rename batch.
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
						tfjsonpath.New("maintenance").AtMapKey("update_inventory"),
						knownvalue.Bool(true),
					),
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("maintenance").AtMapKey("verify_startup_disk"),
						knownvalue.Bool(true),
					),
				},
			},
			{
				Config: policyConfigMaintenance(name, false),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("maintenance").AtMapKey("update_inventory"),
						knownvalue.Bool(false),
					),
				},
			},
		},
	})
}

func policyConfigFilesProcesses(name, searchByPath, executeCommand string) string {
	return fmt.Sprintf(`
resource "jamfplatform_pro_policy" "test" {
  general = {
    name = %q
  }
  files_processes = {
    search_by_path         = %q
    delete_file_if_found   = true
    search_by_filename     = "tf-acc-search-filename"
    update_locate_database = true
    search_by_spotlight    = "tf-acc-spotlight"
    search_for_process     = "tf-acc-process"
    kill_process_if_found  = true
    execute_command        = %q
  }
}
`, name, searchByPath, executeCommand)
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
						tfjsonpath.New("files_processes").AtMapKey("delete_file_if_found"),
						knownvalue.Bool(true),
					),
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("files_processes").AtMapKey("search_by_filename"),
						knownvalue.StringExact("tf-acc-search-filename"),
					),
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("files_processes").AtMapKey("kill_process_if_found"),
						knownvalue.Bool(true),
					),
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("files_processes").AtMapKey("execute_command"),
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
						tfjsonpath.New("files_processes").AtMapKey("execute_command"),
						knownvalue.StringExact("echo tf-acc-updated"),
					),
				},
			},
		},
	})
}

func policyConfigUserInteractionDate(name, startMessage, untilUTC string) string {
	return fmt.Sprintf(`
resource "jamfplatform_pro_policy" "test" {
  general = {
    name = %q
  }
  user_interaction = {
    start_message      = %q
    deferral_type      = "date"
    deferral_until_utc = %q
    complete_message   = "tf-acc complete_message"
  }
}
`, name, startMessage, untilUTC)
}

func policyConfigUserInteractionDuration(name, startMessage string, days int) string {
	return fmt.Sprintf(`
resource "jamfplatform_pro_policy" "test" {
  general = {
    name = %q
  }
  user_interaction = {
    start_message    = %q
    deferral_type    = "duration"
    deferral_days    = %d
    complete_message = "tf-acc complete_message"
  }
}
`, name, startMessage, days)
}

func policyConfigUserInteractionNone(name, startMessage string) string {
	return fmt.Sprintf(`
resource "jamfplatform_pro_policy" "test" {
  general = {
    name = %q
  }
  user_interaction = {
    start_message    = %q
    deferral_type    = "none"
    complete_message = "tf-acc complete_message"
  }
}
`, name, startMessage)
}

// TestAccPolicyResource_UserInteractionFullCoverage exercises every
// user_interaction attribute and every deferral_type variant, including
// in-place transitions between Date / Duration / None. Wire-probe
// 2026-05-27 confirmed the classic /policies PUT accepts these transitions
// without destroy+recreate as long as the provider emits the off-axis wire
// fields with explicit zero values (which buildPolicyUserInteraction does).
func TestAccPolicyResource_UserInteractionFullCoverage(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-policy-userint-" + suffix

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPolicyDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: policyConfigUserInteractionDate(name, "tf-acc start_message initial", "2030-01-01T01:00:00.000+0000"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("user_interaction").AtMapKey("start_message"),
						knownvalue.StringExact("tf-acc start_message initial"),
					),
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("user_interaction").AtMapKey("deferral_type"),
						knownvalue.StringExact("date"),
					),
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("user_interaction").AtMapKey("deferral_until_utc"),
						knownvalue.StringExact("2030-01-01T01:00:00.000+0000"),
					),
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("user_interaction").AtMapKey("complete_message"),
						knownvalue.StringExact("tf-acc complete_message"),
					),
				},
			},
			// In-place Date → Duration transition. Wire probe confirmed the
			// classic API zeroes the off-axis field on PUT.
			{
				Config: policyConfigUserInteractionDuration(name, "tf-acc start_message updated", 3),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("user_interaction").AtMapKey("start_message"),
						knownvalue.StringExact("tf-acc start_message updated"),
					),
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("user_interaction").AtMapKey("deferral_type"),
						knownvalue.StringExact("duration"),
					),
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("user_interaction").AtMapKey("deferral_days"),
						knownvalue.Int64Exact(3),
					),
				},
			},
			// In-place Duration → None transition.
			{
				Config: policyConfigUserInteractionNone(name, "tf-acc start_message no-deferral"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("user_interaction").AtMapKey("deferral_type"),
						knownvalue.StringExact("none"),
					),
				},
			},
		},
	})
}

func policyConfigDateTimeLimitationBad(name, attr, value string) string {
	return fmt.Sprintf(`
resource "jamfplatform_pro_policy" "test" {
  general = {
    name = %q
    date_time_limitations = {
      %s = %q
    }
  }
}
`, name, attr, value)
}

func policyConfigNoExecuteOnBad(name string, days []string) string {
	quoted := make([]string, len(days))
	for i, d := range days {
		quoted[i] = fmt.Sprintf("%q", d)
	}
	return fmt.Sprintf(`
resource "jamfplatform_pro_policy" "test" {
  general = {
    name = %q
    date_time_limitations = {
      no_execute_on = [%s]
    }
  }
}
`, name, strings.Join(quoted, ", "))
}

func policyConfigDeferralUntilUtcBad(name, value string) string {
	return fmt.Sprintf(`
resource "jamfplatform_pro_policy" "test" {
  general = {
    name = %q
  }
  user_interaction = {
    deferral_type      = "date"
    deferral_until_utc = %q
  }
}
`, name, value)
}

// TestAccPolicyResource_DateTimeLimitationsFormatRejection exercises each
// regex validator on the date_time_limitations sub-block. Every step fails
// at plan time with a message keyed off the validator's own error text — no
// API round-trip occurs, so these are fast and credential-cheap once the
// acc framework is up. Wire formats are documented in policy/validators.go.
func TestAccPolicyResource_DateTimeLimitationsFormatRejection(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-policy-dtl-bad-" + suffix

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      policyConfigDateTimeLimitationBad(name, "activation_date", "06/01/2027 14:30:00"),
				ExpectError: regexp.MustCompile(`Value must be 24-hour`),
			},
			{
				Config:      policyConfigDateTimeLimitationBad(name, "activation_date", "2027-06-01T14:30:00"),
				ExpectError: regexp.MustCompile(`Value must be 24-hour`),
			},
			{
				Config:      policyConfigDateTimeLimitationBad(name, "expiration_date", "2027-12-31"),
				ExpectError: regexp.MustCompile(`Value must be 24-hour`),
			},
			{
				Config:      policyConfigDateTimeLimitationBad(name, "no_execute_start", "17:00"),
				ExpectError: regexp.MustCompile(`12-hour`),
			},
			{
				Config:      policyConfigDateTimeLimitationBad(name, "no_execute_start", "5:00 am"),
				ExpectError: regexp.MustCompile(`12-hour`),
			},
			{
				Config:      policyConfigDateTimeLimitationBad(name, "no_execute_end", "13:00 PM"),
				ExpectError: regexp.MustCompile(`12-hour`),
			},
			{
				Config:      policyConfigDateTimeLimitationBad(name, "no_execute_end", "0:00 AM"),
				ExpectError: regexp.MustCompile(`12-hour`),
			},
			{
				Config:      policyConfigNoExecuteOnBad(name, []string{"Sunday"}),
				ExpectError: regexp.MustCompile(`must be one of`),
			},
			{
				Config:      policyConfigNoExecuteOnBad(name, []string{"Sun", "FOO"}),
				ExpectError: regexp.MustCompile(`must be one of`),
			},
		},
	})
}

// TestAccPolicyResource_DeferralUntilUtcFormatRejection exercises the regex
// validator on user_interaction.deferral_until_utc. Wire format is documented
// in policy/validators.go (ISO-8601 with millisecond precision + four-digit
// numeric offset).
func TestAccPolicyResource_DeferralUntilUtcFormatRejection(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-policy-defer-bad-" + suffix

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      policyConfigDeferralUntilUtcBad(name, "2027-01-01T01:00:00Z"),
				ExpectError: regexp.MustCompile(`Value must be ISO-8601`),
			},
			{
				Config:      policyConfigDeferralUntilUtcBad(name, "2027-01-01 01:00:00.000+0000"),
				ExpectError: regexp.MustCompile(`Value must be ISO-8601`),
			},
			{
				Config:      policyConfigDeferralUntilUtcBad(name, "2027-01-01T01:00:00.000+00:00"),
				ExpectError: regexp.MustCompile(`Value must be ISO-8601`),
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
// Update path. action=remediate silently reverts to apply on the server,
// so this test sticks with action=apply throughout.
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
    targets = {
      all_computers = true
      all_jss_users = %t
    }
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
						tfjsonpath.New("scope").AtMapKey("targets").AtMapKey("all_jss_users"),
						knownvalue.Bool(true),
					),
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("scope").AtMapKey("targets").AtMapKey("all_computers"),
						knownvalue.Bool(true),
					),
				},
			},
			{
				Config: policyConfigScopeAllJssUsers(name, false),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("scope").AtMapKey("targets").AtMapKey("all_jss_users"),
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
    targets = {
      computer_ids       = [%q]
      computer_group_ids = [jamfplatform_device_group.fixture.jamf_pro_id]
      building_ids       = [jamfplatform_pro_building.fixture.id]
      department_ids     = [jamfplatform_pro_department.fixture.id]
      user_ids       = [%q]
      user_group_ids = [jamfplatform_pro_user_group.fixture.id]
    }
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
						tfjsonpath.New("scope").AtMapKey("targets").AtMapKey("computer_ids"),
						knownvalue.SetSizeExact(1),
					),
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("scope").AtMapKey("targets").AtMapKey("computer_group_ids"),
						knownvalue.SetSizeExact(1),
					),
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("scope").AtMapKey("targets").AtMapKey("building_ids"),
						knownvalue.SetSizeExact(1),
					),
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("scope").AtMapKey("targets").AtMapKey("department_ids"),
						knownvalue.SetSizeExact(1),
					),
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("scope").AtMapKey("targets").AtMapKey("user_ids"),
						knownvalue.SetSizeExact(1),
					),
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("scope").AtMapKey("targets").AtMapKey("user_group_ids"),
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
    targets = {
      all_computers = true
    }
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
// Their wire round-trip is covered indirectly via the policy 6791 baseline.
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
    targets = {
      all_computers = true
    }
    exclusions = {
      computer_ids                          = [%q]
      computer_group_ids                    = [jamfplatform_device_group.fixture.jamf_pro_id]
      building_ids                          = [jamfplatform_pro_building.fixture.id]
      department_ids                        = [jamfplatform_pro_department.fixture.id]
      user_ids        = [%q]
      user_group_ids  = [jamfplatform_pro_user_group.fixture.id]
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
						tfjsonpath.New("scope").AtMapKey("exclusions").AtMapKey("user_ids"),
						knownvalue.SetSizeExact(1),
					),
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("scope").AtMapKey("exclusions").AtMapKey("user_group_ids"),
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

// policyConfigAccountMaintenanceAccounts_appendFourth is the Step 3
// fixture: identical to Step 2 with a fourth Delete account appended
// at the end of the list. Exercises List growth (List length 3 → 4)
// and verifies the existing three indices are not perturbed.
func policyConfigAccountMaintenanceAccounts_appendFourth(name string) string {
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
      {
        action                            = "Delete"
        username                          = "tf-acc-fourth-delete"
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
// three of the four classic account-maintenance actions in a single policy
// and validates List positional identity across rotation and growth:
//
//   - Create: full account-provisioning attribute set (username, realname,
//     password, home, hint, admin, filevault_enabled, secure_token_allowed).
//   - Reset:  password reset by username.
//   - Delete: removal with `permanently_delete_home_directory = true`. This
//     is the inverted-and-renamed form of the wire field
//     `<archive_home_directory>`. Server receives the inverse on the wire;
//     state stores the UI-canonical semantic.
//
// Step layout: Step 1 creates three accounts. Step 2 rotates the first
// account's `password_wo_version` to prove per-element rotation gating.
// Step 3 appends a fourth Delete account to prove List growth preserves
// positional identity for indices 0–2.
//
// `DisableFileVault` is wired into the schema validator
// (`stringvalidator.OneOf("Create", "Reset", "Delete", "DisableFileVault")`)
// — the wire string is `DisableFileVault` without a trailing `2` despite
// older documentation. The action is NOT exercised in this acceptance test
// because the classic /policies endpoint silently strips
// `<account><action>DisableFileVault</action></account>` entries from new
// policies (round-trip returns no account, framework then reports
// "produced inconsistent result after apply"). The action does survive on
// policies created via the Jamf Pro UI, so the rejection appears to be
// API-only. Manually-probe the precise wire shape needed before adding
// acceptance coverage.
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
			{
				// Step 3: list growth. Append a fourth Delete account; the
				// existing three indices must retain their positional
				// identity and wo_version values.
				Config: policyConfigAccountMaintenanceAccounts_appendFourth(name),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("account_maintenance").AtMapKey("accounts"),
						knownvalue.ListSizeExact(4),
					),
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
						tfjsonpath.New("account_maintenance").AtMapKey("accounts").AtSliceIndex(3).AtMapKey("username"),
						knownvalue.StringExact("tf-acc-fourth-delete"),
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
