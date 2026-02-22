// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

package blueprint_test

import (
	"testing"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/testhelpers"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// testBlueprintConfig returns a helper that builds a blueprint config referencing a smart group.
func testBlueprintConfig(groupConfig string, blueprintBlock string) string {
	return groupConfig + "\n" + blueprintBlock
}

// sharedSmartGroup returns HCL for a reusable smart group used as a blueprint scope target.
const sharedSmartGroup = `
resource "jamfplatform_device_group" "scope" {
	name        = "tf-acc-blueprint-scope"
	group_type  = "smart"
	device_type = "computer"
	criteria = [{
		criteria = "Serial Number"
		operator = "like"
		value    = ""
	}]
}
`

func TestAccResource_Blueprint_CreateAndUpdate(t *testing.T) {
	testhelpers.AccPreCheck(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testBlueprintConfig(sharedSmartGroup, `
					resource "jamfplatform_blueprints_blueprint" "test_update" {
						name          = "tf-acc-create-update-blueprint"
						description   = "Acceptance test — safe to delete"
						deployed      = false
						device_groups = [jamfplatform_device_group.scope.id]

						passcode_policy = {
							require_passcode = true
							minimum_length   = 6
						}
					}
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("jamfplatform_blueprints_blueprint.test_update", "id"),
					resource.TestCheckResourceAttr("jamfplatform_blueprints_blueprint.test_update", "name", "tf-acc-create-update-blueprint"),
					resource.TestCheckResourceAttr("jamfplatform_blueprints_blueprint.test_update", "deployed", "false"),
				),
			},
			{
				Config: testBlueprintConfig(sharedSmartGroup, `
					resource "jamfplatform_blueprints_blueprint" "test_update" {
						name          = "tf-acc-create-update-blueprint-renamed"
						description   = "Updated description"
						deployed      = false
						device_groups = [jamfplatform_device_group.scope.id]

						passcode_policy = {
							require_passcode = true
							minimum_length   = 8
						}
					}
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("jamfplatform_blueprints_blueprint.test_update", "name", "tf-acc-create-update-blueprint-renamed"),
					resource.TestCheckResourceAttr("jamfplatform_blueprints_blueprint.test_update", "description", "Updated description"),
				),
			},
		},
	})
}

func TestAccResource_Blueprint_PasscodePolicy(t *testing.T) {
	testhelpers.AccPreCheck(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testBlueprintConfig(sharedSmartGroup, `
					resource "jamfplatform_blueprints_blueprint" "test_passcode" {
						name          = "tf-acc-passcode-policy"
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
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("jamfplatform_blueprints_blueprint.test_passcode", "id"),
					resource.TestCheckResourceAttr("jamfplatform_blueprints_blueprint.test_passcode", "name", "tf-acc-passcode-policy"),
				),
			},
		},
	})
}

func TestAccResource_Blueprint_MathSettings(t *testing.T) {
	testhelpers.AccPreCheck(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testBlueprintConfig(sharedSmartGroup, `
					resource "jamfplatform_blueprints_blueprint" "test_math" {
						name          = "tf-acc-math-settings"
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
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("jamfplatform_blueprints_blueprint.test_math", "id"),
					resource.TestCheckResourceAttr("jamfplatform_blueprints_blueprint.test_math", "name", "tf-acc-math-settings"),
				),
			},
		},
	})
}

func TestAccResource_Blueprint_AudioAccessorySettings(t *testing.T) {
	testhelpers.AccPreCheck(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testBlueprintConfig(sharedSmartGroup, `
					resource "jamfplatform_blueprints_blueprint" "test_audio" {
						name          = "tf-acc-audio-accessory"
						description   = "Acceptance test — safe to delete"
						deployed      = false
						device_groups = [jamfplatform_device_group.scope.id]

						audio_accessory_settings = {
							temporary_pairing_disabled = true
							unpairing_time_policy      = "Hour"
							unpairing_time_hour        = 14
						}
					}
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("jamfplatform_blueprints_blueprint.test_audio", "id"),
					resource.TestCheckResourceAttr("jamfplatform_blueprints_blueprint.test_audio", "name", "tf-acc-audio-accessory"),
				),
			},
		},
	})
}

func TestAccResource_Blueprint_DiskManagement(t *testing.T) {
	testhelpers.AccPreCheck(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testBlueprintConfig(sharedSmartGroup, `
					resource "jamfplatform_blueprints_blueprint" "test_disk" {
						name          = "tf-acc-disk-management"
						description   = "Acceptance test — safe to delete"
						deployed      = false
						device_groups = [jamfplatform_device_group.scope.id]

						disk_management_settings = {
							external_storage = "ReadOnly"
							network_storage  = "Disallowed"
						}
					}
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("jamfplatform_blueprints_blueprint.test_disk", "id"),
					resource.TestCheckResourceAttr("jamfplatform_blueprints_blueprint.test_disk", "name", "tf-acc-disk-management"),
				),
			},
		},
	})
}

func TestAccResource_Blueprint_SafariSettings(t *testing.T) {
	testhelpers.AccPreCheck(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testBlueprintConfig(sharedSmartGroup, `
					resource "jamfplatform_blueprints_blueprint" "test_safari" {
						name          = "tf-acc-safari-settings"
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
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("jamfplatform_blueprints_blueprint.test_safari", "id"),
					resource.TestCheckResourceAttr("jamfplatform_blueprints_blueprint.test_safari", "name", "tf-acc-safari-settings"),
				),
			},
		},
	})
}

func TestAccResource_Blueprint_SoftwareUpdateSettings(t *testing.T) {
	testhelpers.AccPreCheck(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testBlueprintConfig(sharedSmartGroup, `
					resource "jamfplatform_blueprints_blueprint" "test_swu_settings" {
						name          = "tf-acc-sw-update-settings"
						description   = "Acceptance test — safe to delete"
						deployed      = false
						device_groups = [jamfplatform_device_group.scope.id]

						software_update_settings = {
							allow_standard_user_os_updates           = true
							automatic_download                       = "AlwaysOn"
							automatic_install_os_updates             = "AlwaysOn"
							automatic_install_security_updates       = "AlwaysOn"
							deferral_combined_period_days            = "7"
							deferral_major_period_days               = "30"
							deferral_minor_period_days               = "14"
							deferral_system_period_days              = "3"
							notifications_enabled                    = true
							rapid_security_response_enabled          = true
							rapid_security_response_rollback_enabled = false
							recommended_cadence                      = "Newest"
						}
					}
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("jamfplatform_blueprints_blueprint.test_swu_settings", "id"),
					resource.TestCheckResourceAttr("jamfplatform_blueprints_blueprint.test_swu_settings", "name", "tf-acc-sw-update-settings"),
				),
			},
		},
	})
}

func TestAccResource_Blueprint_SoftwareUpdate_Automatic(t *testing.T) {
	testhelpers.AccPreCheck(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testBlueprintConfig(sharedSmartGroup, `
					resource "jamfplatform_blueprints_blueprint" "test_swu_auto" {
						name          = "tf-acc-sw-update-auto"
						description   = "Acceptance test — safe to delete"
						deployed      = false
						device_groups = [jamfplatform_device_group.scope.id]

						software_update = {
							ignore_major_versions = true
							deployment_time       = "02:00"
							enforce_after_days    = 7
						}
					}
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("jamfplatform_blueprints_blueprint.test_swu_auto", "id"),
					resource.TestCheckResourceAttr("jamfplatform_blueprints_blueprint.test_swu_auto", "name", "tf-acc-sw-update-auto"),
				),
			},
		},
	})
}

func TestAccResource_Blueprint_LegacyPayloads(t *testing.T) {
	testhelpers.AccPreCheck(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testBlueprintConfig(sharedSmartGroup, `
					resource "jamfplatform_blueprints_blueprint" "test_legacy" {
						name          = "tf-acc-legacy-payloads"
						description   = "Acceptance test — safe to delete"
						deployed      = false
						device_groups = [jamfplatform_device_group.scope.id]

						legacy_payloads = jsonencode([
							{
								allowSafariHistoryClearing = false
								allowSafariPrivateBrowsing = false
								payloadType                = "com.apple.applicationaccess"
								payloadIdentifier          = "tf-acc-test-payload-001"
							}
						])
					}
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("jamfplatform_blueprints_blueprint.test_legacy", "id"),
					resource.TestCheckResourceAttr("jamfplatform_blueprints_blueprint.test_legacy", "name", "tf-acc-legacy-payloads"),
				),
			},
		},
	})
}

func TestAccResource_Blueprint_CustomDeclarations(t *testing.T) {
	testhelpers.AccPreCheck(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testBlueprintConfig(sharedSmartGroup, `
					resource "jamfplatform_blueprints_blueprint" "test_custom" {
						name          = "tf-acc-custom-declarations"
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
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("jamfplatform_blueprints_blueprint.test_custom", "id"),
					resource.TestCheckResourceAttr("jamfplatform_blueprints_blueprint.test_custom", "name", "tf-acc-custom-declarations"),
				),
			},
		},
	})
}

func TestAccResource_Blueprint_SafariBookmarks(t *testing.T) {
	testhelpers.AccPreCheck(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testBlueprintConfig(sharedSmartGroup, `
					resource "jamfplatform_blueprints_blueprint" "test_bookmarks" {
						name          = "tf-acc-safari-bookmarks"
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
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("jamfplatform_blueprints_blueprint.test_bookmarks", "id"),
					resource.TestCheckResourceAttr("jamfplatform_blueprints_blueprint.test_bookmarks", "name", "tf-acc-safari-bookmarks"),
				),
			},
		},
	})
}

func TestAccResource_Blueprint_SafariExtensions(t *testing.T) {
	testhelpers.AccPreCheck(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testBlueprintConfig(sharedSmartGroup, `
					resource "jamfplatform_blueprints_blueprint" "test_extensions" {
						name          = "tf-acc-safari-extensions"
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
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("jamfplatform_blueprints_blueprint.test_extensions", "id"),
					resource.TestCheckResourceAttr("jamfplatform_blueprints_blueprint.test_extensions", "name", "tf-acc-safari-extensions"),
				),
			},
		},
	})
}

func TestAccDataSource_Blueprint(t *testing.T) {
	testhelpers.AccPreCheck(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testBlueprintConfig(sharedSmartGroup, `
					resource "jamfplatform_blueprints_blueprint" "source" {
						name          = "tf-acc-ds-blueprint"
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
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.jamfplatform_blueprints_blueprint.test", "name", "tf-acc-ds-blueprint"),
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
