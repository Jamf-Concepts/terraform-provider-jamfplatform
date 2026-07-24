// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

package blueprint_test

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	bpSDK "github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/blueprints"
	dgSDK "github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/devicegroups"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/testhelpers"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// testBlueprintConfig returns a helper that builds a blueprint config referencing a smart group.
func testBlueprintConfig(groupConfig string, blueprintBlock string) string {
	return groupConfig + "\n" + blueprintBlock
}

// smartGroupHCL returns HCL for a smart group with a unique name derived from the test suffix
// and the run-wide suffix.
func smartGroupHCL(testSuffix string) string {
	return fmt.Sprintf(`
resource "jamfplatform_device_group" "scope" {
	name        = "tf-acc-bp-scope-%s-%s"
	group_type  = "smart"
	device_type = "computer"
	criteria = [{
		criteria = "Serial Number"
		operator = "like"
		value    = ""
	}]
}
`, testSuffix, testhelpers.RunSuffix())
}

// testAccCheckBlueprintResourcesDestroy verifies that blueprints and device groups
// created during the test have been destroyed.
func testAccCheckBlueprintResourcesDestroy(t *testing.T) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		c := testhelpers.NewAcceptanceClient(t)
		bpClient := bpSDK.New(c)
		dgClient := dgSDK.New(c)
		ctx := context.Background()

		for _, rs := range s.RootModule().Resources {
			switch rs.Type {
			case "jamfplatform_blueprints_blueprint":
				_, err := bpClient.GetBlueprint(ctx, rs.Primary.ID)
				if err == nil {
					return fmt.Errorf("blueprint %s still exists after destroy", rs.Primary.ID)
				}
				if !helpers.IsNotFoundError(err) {
					return fmt.Errorf("error checking blueprint %s: %s", rs.Primary.ID, err)
				}
			case "jamfplatform_device_group":
				deadline := time.Now().Add(60 * time.Second)
				for time.Now().Before(deadline) {
					_, err := dgClient.GetDeviceGroup(ctx, rs.Primary.ID)
					if err != nil {
						if helpers.IsNotFoundError(err) {
							break
						}
						return fmt.Errorf("error checking device group %s: %s", rs.Primary.ID, err)
					}
					time.Sleep(2 * time.Second)
				}
			}
		}
		return nil
	}
}

func TestAccResource_Blueprint_CreateAndUpdate(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-create-update-blueprint-" + suffix
	nameRenamed := "tf-acc-create-update-blueprint-renamed-" + suffix

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckBlueprintResourcesDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: testBlueprintConfig(smartGroupHCL("update"), fmt.Sprintf(`
					resource "jamfplatform_blueprints_blueprint" "test_update" {
						name          = %q
						description   = "Acceptance test — safe to delete"
						deployed      = false
						device_groups = [jamfplatform_device_group.scope.id]

						component_blocks = [
							{
								name = "Passcode Policy"
								passcode_policy = {
									require_passcode = true
									minimum_length   = 6
								}
							},
						]
					}
				`, name)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("jamfplatform_blueprints_blueprint.test_update", "id"),
					resource.TestCheckResourceAttr("jamfplatform_blueprints_blueprint.test_update", "name", name),
					resource.TestCheckResourceAttr("jamfplatform_blueprints_blueprint.test_update", "deployed", "false"),
					resource.TestCheckResourceAttr("jamfplatform_blueprints_blueprint.test_update", "component_blocks.#", "1"),
					resource.TestCheckResourceAttr("jamfplatform_blueprints_blueprint.test_update", "component_blocks.0.name", "Passcode Policy"),
					resource.TestCheckResourceAttr("jamfplatform_blueprints_blueprint.test_update", "component_blocks.0.passcode_policy.minimum_length", "6"),
				),
			},
			{
				Config: testBlueprintConfig(smartGroupHCL("update"), fmt.Sprintf(`
					resource "jamfplatform_blueprints_blueprint" "test_update" {
						name          = %q
						description   = "Updated description"
						deployed      = false
						device_groups = [jamfplatform_device_group.scope.id]

						component_blocks = [
							{
								name = "Passcode Policy"
								passcode_policy = {
									require_passcode = true
									minimum_length   = 8
								}
							},
						]
					}
				`, nameRenamed)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("jamfplatform_blueprints_blueprint.test_update", "name", nameRenamed),
					resource.TestCheckResourceAttr("jamfplatform_blueprints_blueprint.test_update", "description", "Updated description"),
					resource.TestCheckResourceAttr("jamfplatform_blueprints_blueprint.test_update", "component_blocks.0.passcode_policy.minimum_length", "8"),
				),
			},
		},
	})
}

// checkActivationConditionsContainGroupID verifies the first block's activation_conditions
// expression embeds the managed device group's ID verbatim, proving Terraform interpolation of a
// device-group reference round-trips through the API.
func checkActivationConditionsContainGroupID(blueprintRes, groupRes string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		group, ok := s.RootModule().Resources[groupRes]
		if !ok {
			return fmt.Errorf("device group resource %q not found in state", groupRes)
		}
		blueprint, ok := s.RootModule().Resources[blueprintRes]
		if !ok {
			return fmt.Errorf("blueprint resource %q not found in state", blueprintRes)
		}
		groupID := group.Primary.ID
		conditions := blueprint.Primary.Attributes["component_blocks.0.activation_conditions"]
		if !strings.Contains(conditions, groupID) {
			return fmt.Errorf("component_blocks.0.activation_conditions %q does not contain device group id %q", conditions, groupID)
		}
		return nil
	}
}

func TestAccResource_Blueprint_ActivationConditions(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-activation-conditions-" + suffix
	res := "jamfplatform_blueprints_blueprint.test_activation"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckBlueprintResourcesDestroy(t),
		Steps: []resource.TestStep{
			// Step 1: a block's activation_conditions references the managed device group by id via
			// Terraform interpolation; the resolved expression must round-trip with the group's UUID
			// embedded verbatim.
			{
				Config: testBlueprintConfig(smartGroupHCL("activation"), fmt.Sprintf(`
					resource "jamfplatform_blueprints_blueprint" "test_activation" {
						name          = %q
						description   = "Acceptance test — safe to delete"
						deployed      = false
						device_groups = [jamfplatform_device_group.scope.id]

						component_blocks = [
							{
								name                  = "Shared iPad Software Updates"
								activation_conditions = "ANY @property(jamf.device.groups) IN {'${jamfplatform_device_group.scope.id}'} AND @status(device.model.family) == 'iPad'"
								software_update = {
									ignore_major_versions = true
									deployment_time       = "02:00"
									enforce_after_days    = 7
								}
							},
						]
					}
				`, name)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(res, "id"),
					resource.TestMatchResourceAttr(
						res,
						"component_blocks.0.activation_conditions",
						regexp.MustCompile(`^ANY @property\(jamf\.device\.groups\) IN \{'[0-9a-fA-F-]{36}'\} AND @status\(device\.model\.family\) == 'iPad'$`),
					),
					checkActivationConditionsContainGroupID(res, "jamfplatform_device_group.scope"),
				),
			},
			// Step 2: replace with a static expression; verifies the update path round-trips the
			// value byte-for-byte (no server-side normalization).
			{
				Config: testBlueprintConfig(smartGroupHCL("activation"), fmt.Sprintf(`
					resource "jamfplatform_blueprints_blueprint" "test_activation" {
						name          = %q
						description   = "Acceptance test — safe to delete"
						deployed      = false
						device_groups = [jamfplatform_device_group.scope.id]

						component_blocks = [
							{
								name                  = "Shared iPad Software Updates"
								activation_conditions = "@status(device.model.family) == 'iPad'"
								software_update = {
									ignore_major_versions = true
									deployment_time       = "02:00"
									enforce_after_days    = 7
								}
							},
						]
					}
				`, name)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(res, "component_blocks.0.activation_conditions", "@status(device.model.family) == 'iPad'"),
				),
			},
			// Step 3: drop the block's activation_conditions entirely; it must clear back to null.
			{
				Config: testBlueprintConfig(smartGroupHCL("activation"), fmt.Sprintf(`
					resource "jamfplatform_blueprints_blueprint" "test_activation" {
						name          = %q
						description   = "Acceptance test — safe to delete"
						deployed      = false
						device_groups = [jamfplatform_device_group.scope.id]

						component_blocks = [
							{
								name = "Shared iPad Software Updates"
								software_update = {
									ignore_major_versions = true
									deployment_time       = "02:00"
									enforce_after_days    = 7
								}
							},
						]
					}
				`, name)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr(res, "component_blocks.0.activation_conditions"),
				),
			},
		},
	})
}

func TestAccResource_Blueprint_PasscodePolicy(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-passcode-policy-" + suffix

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckBlueprintResourcesDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: testBlueprintConfig(smartGroupHCL("passcode"), fmt.Sprintf(`
					resource "jamfplatform_blueprints_blueprint" "test_passcode" {
						name          = %q
						description   = "Acceptance test — safe to delete"
						deployed      = false
						device_groups = [jamfplatform_device_group.scope.id]

						component_blocks = [
							{
								name = "Passcode Policy"
								passcode_policy = {
									require_passcode              = true
									minimum_length                = 8
									maximum_failed_attempts       = 5
									maximum_inactivity_in_minutes = 10
								}
							},
						]
					}
				`, name)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("jamfplatform_blueprints_blueprint.test_passcode", "id"),
					resource.TestCheckResourceAttr("jamfplatform_blueprints_blueprint.test_passcode", "name", name),
					resource.TestCheckResourceAttr("jamfplatform_blueprints_blueprint.test_passcode", "component_blocks.0.passcode_policy.minimum_length", "8"),
				),
			},
		},
	})
}

func TestAccResource_Blueprint_MathSettings(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-math-settings-" + suffix

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckBlueprintResourcesDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: testBlueprintConfig(smartGroupHCL("math"), fmt.Sprintf(`
					resource "jamfplatform_blueprints_blueprint" "test_math" {
						name          = %q
						description   = "Acceptance test — safe to delete"
						deployed      = false
						device_groups = [jamfplatform_device_group.scope.id]

						component_blocks = [
							{
								name = "Math Settings"
								math_settings = {
									calculator_basic_mode_add_square_root  = true
									calculator_scientific_mode_enabled     = true
									calculator_programmer_mode_enabled     = false
									calculator_math_notes_mode_enabled     = true
									calculator_input_modes_unit_conversion = true
									calculator_input_modes_rpn             = false
									system_behavior_keyboard_suggestions   = true
									system_behavior_math_notes             = true
								}
							},
						]
					}
				`, name)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("jamfplatform_blueprints_blueprint.test_math", "id"),
					resource.TestCheckResourceAttr("jamfplatform_blueprints_blueprint.test_math", "name", name),
					resource.TestCheckResourceAttr("jamfplatform_blueprints_blueprint.test_math", "component_blocks.0.math_settings.calculator_scientific_mode_enabled", "true"),
				),
			},
		},
	})
}

func TestAccResource_Blueprint_AudioAccessorySettings(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-audio-accessory-" + suffix

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckBlueprintResourcesDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: testBlueprintConfig(smartGroupHCL("audio"), fmt.Sprintf(`
					resource "jamfplatform_blueprints_blueprint" "test_audio" {
						name          = %q
						description   = "Acceptance test — safe to delete"
						deployed      = false
						device_groups = [jamfplatform_device_group.scope.id]

						component_blocks = [
							{
								name = "Audio Accessory Settings"
								audio_accessory_settings = {
									temporary_pairing_disabled = true
									unpairing_time_policy      = "Hour"
									unpairing_time_hour        = 14
								}
							},
						]
					}
				`, name)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("jamfplatform_blueprints_blueprint.test_audio", "id"),
					resource.TestCheckResourceAttr("jamfplatform_blueprints_blueprint.test_audio", "name", name),
					resource.TestCheckResourceAttr("jamfplatform_blueprints_blueprint.test_audio", "component_blocks.0.audio_accessory_settings.unpairing_time_hour", "14"),
				),
			},
		},
	})
}

func TestAccResource_Blueprint_DiskManagement(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-disk-management-" + suffix

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckBlueprintResourcesDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: testBlueprintConfig(smartGroupHCL("disk"), fmt.Sprintf(`
					resource "jamfplatform_blueprints_blueprint" "test_disk" {
						name          = %q
						description   = "Acceptance test — safe to delete"
						deployed      = false
						device_groups = [jamfplatform_device_group.scope.id]

						component_blocks = [
							{
								name = "Disk Management"
								disk_management_settings = {
									external_storage = "ReadOnly"
									network_storage  = "Disallowed"
								}
							},
						]
					}
				`, name)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("jamfplatform_blueprints_blueprint.test_disk", "id"),
					resource.TestCheckResourceAttr("jamfplatform_blueprints_blueprint.test_disk", "name", name),
				),
			},
		},
	})
}

func TestAccResource_Blueprint_SafariSettings(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-safari-settings-" + suffix

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckBlueprintResourcesDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: testBlueprintConfig(smartGroupHCL("safari"), fmt.Sprintf(`
					resource "jamfplatform_blueprints_blueprint" "test_safari" {
						name          = %q
						description   = "Acceptance test — safe to delete"
						deployed      = false
						device_groups = [jamfplatform_device_group.scope.id]

						component_blocks = [
							{
								name = "Safari Settings"
								safari_settings = {
									accept_cookies         = "VisitedWebsites"
									allow_private_browsing = false
									allow_javascript       = true
									allow_popups           = false
									allow_history_clearing = false
								}
							},
						]
					}
				`, name)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("jamfplatform_blueprints_blueprint.test_safari", "id"),
					resource.TestCheckResourceAttr("jamfplatform_blueprints_blueprint.test_safari", "name", name),
				),
			},
		},
	})
}

func TestAccResource_Blueprint_SoftwareUpdateSettings(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-sw-update-settings-" + suffix

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckBlueprintResourcesDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: testBlueprintConfig(smartGroupHCL("swu-settings"), fmt.Sprintf(`
					resource "jamfplatform_blueprints_blueprint" "test_swu_settings" {
						name          = %q
						description   = "Acceptance test — safe to delete"
						deployed      = false
						device_groups = [jamfplatform_device_group.scope.id]

						component_blocks = [
							{
								name = "Software Update Settings"
								software_update_settings = {
									allow_standard_user_os_updates           = true
									automatic_download                       = "AlwaysOn"
									automatic_install_os_updates             = "AlwaysOn"
									automatic_install_security_updates       = "AlwaysOn"
									deferral_combined_period_days            = 7
									deferral_major_period_days               = 30
									deferral_minor_period_days               = 14
									deferral_system_period_days              = 3
									notifications_enabled                    = true
									rapid_security_response_enabled          = true
									rapid_security_response_rollback_enabled = false
									recommended_cadence                      = "Newest"
								}
							},
						]
					}
				`, name)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("jamfplatform_blueprints_blueprint.test_swu_settings", "id"),
					resource.TestCheckResourceAttr("jamfplatform_blueprints_blueprint.test_swu_settings", "name", name),
				),
			},
		},
	})
}

func TestAccResource_Blueprint_SoftwareUpdate_Automatic(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-sw-update-auto-" + suffix

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckBlueprintResourcesDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: testBlueprintConfig(smartGroupHCL("swu-auto"), fmt.Sprintf(`
					resource "jamfplatform_blueprints_blueprint" "test_swu_auto" {
						name          = %q
						description   = "Acceptance test — safe to delete"
						deployed      = false
						device_groups = [jamfplatform_device_group.scope.id]

						component_blocks = [
							{
								name = "Latest OS Software Updates"
								software_update = {
									ignore_major_versions = true
									deployment_time       = "02:00"
									enforce_after_days    = 7
								}
							},
						]
					}
				`, name)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("jamfplatform_blueprints_blueprint.test_swu_auto", "id"),
					resource.TestCheckResourceAttr("jamfplatform_blueprints_blueprint.test_swu_auto", "name", name),
				),
			},
		},
	})
}

func TestAccResource_Blueprint_LegacyPayloads(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-legacy-payloads-" + suffix

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckBlueprintResourcesDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: testBlueprintConfig(smartGroupHCL("legacy"), fmt.Sprintf(`
					resource "jamfplatform_blueprints_blueprint" "test_legacy" {
						name          = %q
						description   = "Acceptance test — safe to delete"
						deployed      = false
						device_groups = [jamfplatform_device_group.scope.id]

						component_blocks = [
							{
								name = "Safari Restrictions"
								legacy_payloads = [
									{
										payload_type = "com.apple.applicationaccess"
										settings = jsonencode({
											allowSafariHistoryClearing = false
											allowSafariPrivateBrowsing = false
										})
									}
								]
							},
						]
					}
				`, name)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("jamfplatform_blueprints_blueprint.test_legacy", "id"),
					resource.TestCheckResourceAttr("jamfplatform_blueprints_blueprint.test_legacy", "name", name),
				),
			},
		},
	})
}

// TestAccResource_Blueprint_LegacyPayloads_NullFields_PlanStability is a regression test for issue
// #282, adapted to component_blocks. A legacy payload whose settings contain explicit nulls (here
// com.apple.notificationsettings) previously produced a perpetual in-place update, and applying that
// update failed with "Provider produced inconsistent result after apply" on the server-stamped
// `updated` attribute. Step 1 creates the blueprint; step 2 re-applies the identical config and
// asserts an empty plan (the dynamic null-typing reconcile); step 3 makes a genuine metadata change
// (description) that forces an in-place update and must apply without an inconsistent-result error
// (ModifyPlan marks `updated` unknown); step 4 confirms plan stability after that update; step 5
// makes a genuine change to the payload itself, proving the reconcile surfaces real payload edits
// rather than masking them; step 6 confirms stability again.
func TestAccResource_Blueprint_LegacyPayloads_NullFields_PlanStability(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-legacy-null-" + suffix
	addr := "jamfplatform_blueprints_blueprint.test_legacy_null"

	config := func(description string, firstNotificationsEnabled bool) string {
		return testBlueprintConfig(smartGroupHCL("legacynull"), fmt.Sprintf(`
			resource "jamfplatform_blueprints_blueprint" "test_legacy_null" {
				name          = %q
				description   = %q
				deployed      = false
				device_groups = [jamfplatform_device_group.scope.id]

				component_blocks = [
					{
						name = "Notification Settings"
						legacy_payloads = [
							{
								payload_type = "com.apple.notificationsettings"
								settings = jsonencode({
									NotificationSettings = [
										{
											AlertType                = 0
											PreviewType              = null
											GroupingType             = null
											BadgesEnabled            = null
											ShowInCarPlay            = null
											SoundsEnabled            = null
											BundleIdentifier         = "_SYSTEM_CENTER_:com.apple.followup.alert"
											ShowInLockScreen         = false
											CriticalAlertEnabled     = null
											NotificationsEnabled     = %t
											ShowInNotificationCenter = false
										},
										{
											AlertType                = 2
											PreviewType              = null
											GroupingType             = null
											BadgesEnabled            = true
											ShowInCarPlay            = null
											SoundsEnabled            = true
											BundleIdentifier         = "au.bartreardon.dialog"
											ShowInLockScreen         = false
											CriticalAlertEnabled     = true
											NotificationsEnabled     = true
											ShowInNotificationCenter = true
										},
									]
								})
							},
						]
					},
				]
			}
		`, name, description, firstNotificationsEnabled))
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckBlueprintResourcesDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: config("Acceptance test — safe to delete", false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(addr, "id"),
					resource.TestCheckResourceAttr(addr, "name", name),
					resource.TestCheckResourceAttrSet(addr, "updated"),
				),
			},
			{
				Config: config("Acceptance test — safe to delete", false),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
			{
				Config: config("Updated description", false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(addr, "description", "Updated description"),
				),
			},
			{
				Config: config("Updated description", false),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
			{
				Config: config("Updated description", true),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectNonEmptyPlan()},
				},
			},
			{
				Config: config("Updated description", true),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
		},
	})
}

func TestAccResource_Blueprint_CustomDeclarations(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-custom-declarations-" + suffix

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckBlueprintResourcesDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: testBlueprintConfig(smartGroupHCL("custom"), fmt.Sprintf(`
					resource "jamfplatform_blueprints_blueprint" "test_custom" {
						name          = %q
						description   = "Acceptance test — safe to delete"
						deployed      = false
						device_groups = [jamfplatform_device_group.scope.id]

						component_blocks = [
							{
								name = "Custom Declarations"
								custom_declarations = {
									declaration = [{
										channel = "SYSTEM"
										kind    = "CONFIGURATION"
										type    = "com.apple.configuration.softwareupdate.settings"
										payload = jsonencode({
											Beta = {
												ProgramEnrollment = "AlwaysOff"
											}
										})
									}]
								}
							},
						]
					}
				`, name)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("jamfplatform_blueprints_blueprint.test_custom", "id"),
					resource.TestCheckResourceAttr("jamfplatform_blueprints_blueprint.test_custom", "name", name),
				),
			},
		},
	})
}

func TestAccResource_Blueprint_SafariBookmarks(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-safari-bookmarks-" + suffix

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckBlueprintResourcesDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: testBlueprintConfig(smartGroupHCL("bookmarks"), fmt.Sprintf(`
					resource "jamfplatform_blueprints_blueprint" "test_bookmarks" {
						name          = %q
						description   = "Acceptance test — safe to delete"
						deployed      = false
						device_groups = [jamfplatform_device_group.scope.id]

						component_blocks = [
							{
								name = "Safari Bookmarks"
								safari_bookmarks = {
									managed_bookmarks = [{
										group_identifier = "tf-acc-group-1"
										title            = "Test Bookmarks"
										bookmarks = [{
											type  = "bookmark"
											title = "Jamf"
											url   = "https://www.jamf.com"
										}]
									}]
								}
							},
						]
					}
				`, name)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("jamfplatform_blueprints_blueprint.test_bookmarks", "id"),
					resource.TestCheckResourceAttr("jamfplatform_blueprints_blueprint.test_bookmarks", "name", name),
				),
			},
		},
	})
}

func TestAccResource_Blueprint_SafariExtensions(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-safari-extensions-" + suffix

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckBlueprintResourcesDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: testBlueprintConfig(smartGroupHCL("extensions"), fmt.Sprintf(`
					resource "jamfplatform_blueprints_blueprint" "test_extensions" {
						name          = %q
						description   = "Acceptance test — safe to delete"
						deployed      = false
						device_groups = [jamfplatform_device_group.scope.id]

						component_blocks = [
							{
								name = "Safari Extensions"
								safari_extensions = {
									managed_extensions = [{
										extension_id     = "com.example.test.extension (ABC1234567)"
										state            = "Allowed"
										private_browsing = "AlwaysOff"
									}]
								}
							},
						]
					}
				`, name)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("jamfplatform_blueprints_blueprint.test_extensions", "id"),
					resource.TestCheckResourceAttr("jamfplatform_blueprints_blueprint.test_extensions", "name", name),
				),
			},
		},
	})
}

// TestAccResource_Blueprint_DeprecatedFlatMode keeps a thin regression on the deprecated top-level
// authoring style, which continues to function during its removal window (see the
// PLATFORM-DEPRECATED marker in resource.go). It proves a flat single-component blueprint still
// applies and reads back cleanly, and that a re-apply is a no-op.
func TestAccResource_Blueprint_DeprecatedFlatMode(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-flat-deprecated-" + suffix
	addr := "jamfplatform_blueprints_blueprint.test_flat"

	config := testBlueprintConfig(smartGroupHCL("flat"), fmt.Sprintf(`
		resource "jamfplatform_blueprints_blueprint" "test_flat" {
			name          = %q
			description   = "Acceptance test — safe to delete"
			deployed      = false
			device_groups = [jamfplatform_device_group.scope.id]

			passcode_policy = {
				require_passcode = true
				minimum_length   = 6
			}
		}
	`, name))

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckBlueprintResourcesDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(addr, "id"),
					resource.TestCheckResourceAttr(addr, "passcode_policy.minimum_length", "6"),
					resource.TestCheckNoResourceAttr(addr, "component_blocks.#"),
				),
			},
			{
				Config: config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
		},
	})
}

func TestAccDataSource_Blueprint(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-ds-blueprint-" + suffix

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckBlueprintResourcesDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: testBlueprintConfig(smartGroupHCL("ds"), fmt.Sprintf(`
					resource "jamfplatform_blueprints_blueprint" "source" {
						name          = %q
						description   = "Acceptance test — safe to delete"
						deployed      = false
						device_groups = [jamfplatform_device_group.scope.id]

						component_blocks = [
							{
								name = "Passcode Policy"
								passcode_policy = {
									require_passcode = true
									minimum_length   = 6
								}
							},
						]
					}

					data "jamfplatform_blueprints_blueprint" "test" {
						name = jamfplatform_blueprints_blueprint.source.name
					}
				`, name)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.jamfplatform_blueprints_blueprint.test", "name", name),
					resource.TestCheckResourceAttrSet("data.jamfplatform_blueprints_blueprint.test", "blueprint_id"),
					resource.TestCheckResourceAttr("data.jamfplatform_blueprints_blueprint.test", "component_blocks.#", "1"),
					resource.TestCheckResourceAttr("data.jamfplatform_blueprints_blueprint.test", "component_blocks.0.name", "Passcode Policy"),
					resource.TestCheckResourceAttrSet("data.jamfplatform_blueprints_blueprint.test", "component_blocks.0.component.#"),
				),
			},
		},
	})
}

func TestAccDataSource_Blueprints(t *testing.T) {
	testhelpers.AccPreCheck(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckBlueprintResourcesDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: `
					data "jamfplatform_blueprints" "all" {}
				`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.jamfplatform_blueprints.all", "blueprints.#"),
				),
			},
		},
	})
}

func TestAccDataSource_BlueprintComponents(t *testing.T) {
	testhelpers.AccPreCheck(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckBlueprintResourcesDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: `
					data "jamfplatform_blueprints_components" "all" {}
				`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.jamfplatform_blueprints_components.all", "components.#"),
				),
			},
		},
	})
}
