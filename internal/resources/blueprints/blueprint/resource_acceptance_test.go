// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

package blueprint_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/testhelpers"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
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
		ctx := context.Background()

		for _, rs := range s.RootModule().Resources {
			switch rs.Type {
			case "jamfplatform_blueprints_blueprint":
				_, err := c.GetBlueprint(ctx, rs.Primary.ID)
				if err == nil {
					return fmt.Errorf("blueprint %s still exists after destroy", rs.Primary.ID)
				}
				if !helpers.IsNotFoundError(err) {
					return fmt.Errorf("error checking blueprint %s: %s", rs.Primary.ID, err)
				}
			case "jamfplatform_device_group":
				deadline := time.Now().Add(60 * time.Second)
				for time.Now().Before(deadline) {
					_, err := c.GetDeviceGroup(ctx, rs.Primary.ID)
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

						passcode_policy = {
							require_passcode = true
							minimum_length   = 6
						}
					}
				`, name)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("jamfplatform_blueprints_blueprint.test_update", "id"),
					resource.TestCheckResourceAttr("jamfplatform_blueprints_blueprint.test_update", "name", name),
					resource.TestCheckResourceAttr("jamfplatform_blueprints_blueprint.test_update", "deployed", "false"),
				),
			},
			{
				Config: testBlueprintConfig(smartGroupHCL("update"), fmt.Sprintf(`
					resource "jamfplatform_blueprints_blueprint" "test_update" {
						name          = %q
						description   = "Updated description"
						deployed      = false
						device_groups = [jamfplatform_device_group.scope.id]

						passcode_policy = {
							require_passcode = true
							minimum_length   = 8
						}
					}
				`, nameRenamed)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("jamfplatform_blueprints_blueprint.test_update", "name", nameRenamed),
					resource.TestCheckResourceAttr("jamfplatform_blueprints_blueprint.test_update", "description", "Updated description"),
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

						passcode_policy = {
							require_passcode     = true
							minimum_length       = 8
							maximum_failed_attempts = 5
							maximum_inactivity_in_minutes = 10
						}
					}
				`, name)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("jamfplatform_blueprints_blueprint.test_passcode", "id"),
					resource.TestCheckResourceAttr("jamfplatform_blueprints_blueprint.test_passcode", "name", name),
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

						math_settings = {
							calculator_basic_mode_add_square_root = true
							calculator_scientific_mode_enabled    = true
							calculator_programmer_mode_enabled    = false
							calculator_math_notes_mode_enabled    = true
							calculator_input_modes_unit_conversion = true
							calculator_input_modes_rpn             = false
							system_behavior_keyboard_suggestions   = true
							system_behavior_math_notes             = true
						}
					}
				`, name)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("jamfplatform_blueprints_blueprint.test_math", "id"),
					resource.TestCheckResourceAttr("jamfplatform_blueprints_blueprint.test_math", "name", name),
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

						audio_accessory_settings = {
							temporary_pairing_disabled = true
							unpairing_time_policy      = "Hour"
							unpairing_time_hour        = 14
						}
					}
				`, name)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("jamfplatform_blueprints_blueprint.test_audio", "id"),
					resource.TestCheckResourceAttr("jamfplatform_blueprints_blueprint.test_audio", "name", name),
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

						disk_management_settings = {
							external_storage = "ReadOnly"
							network_storage  = "Disallowed"
						}
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

						safari_settings = {
							accept_cookies       = "VisitedWebsites"
							allow_private_browsing = false
							allow_javascript       = true
							allow_popups           = false
							allow_history_clearing = false
						}
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

						software_update = {
							ignore_major_versions = true
							deployment_time       = "02:00"
							enforce_after_days    = 7
						}
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

						legacy_payloads = [
							{
								payload_type = "com.apple.applicationaccess"
								settings = {
									allowSafariHistoryClearing = false
									allowSafariPrivateBrowsing = false
								}
							}
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

						safari_extensions = {
							managed_extensions = [{
								extension_id    = "com.example.test.extension (ABC1234567)"
								state           = "Allowed"
								private_browsing = "AlwaysOff"
							}]
						}
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

						passcode_policy = {
							require_passcode = true
							minimum_length   = 6
						}
					}

					data "jamfplatform_blueprints_blueprint" "test" {
						name = jamfplatform_blueprints_blueprint.source.name
					}
				`, name)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.jamfplatform_blueprints_blueprint.test", "name", name),
					resource.TestCheckResourceAttrSet("data.jamfplatform_blueprints_blueprint.test", "blueprint_id"),
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
