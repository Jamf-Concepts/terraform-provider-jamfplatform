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
