// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

// Tests in this file talk to the Jamf ProClassic /macapplications endpoint.
// Classic has known concurrency issues when multiple writes hit the same
// resource type — keep these tests serial with any future classic acceptance
// work in this package.
//
// Scope happy-paths use all_computers = true so they need no pre-existing
// tenant target objects, and general.site is left unset to avoid the
// site/scope "not site-enabled" 409 invariant. The ungated tests never set
// vpp.assign_vpp_device_based_licenses = true (a non-VPP title 409s "App is
// not available for device assignment"); the gated full-metadata test
// (TestAccResource_ProMacApp_VPPFullMetadata) does, behind JAMFPLATFORM_VPP_TOKEN.

package mac_app_store_app_test

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/proclassic"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/querycheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/testhelpers"
)

const macAppResourceAddr = "jamfplatform_pro_mac_app_store_app.test"

// vppTokenEnvVar holds the base64 `.vpptoken` contents (same gate as the
// volume_purchasing_location acceptance test). The full-metadata VPP test
// below is skipped unless it is set, because device-based VPP assignment
// requires a real Apple Business Manager / Apple School Manager token whose
// location owns device-assignable licenses for the test title (Jamf Parent).
// Tokens MUST come from env — never commit token material.
const vppTokenEnvVar = "JAMFPLATFORM_VPP_TOKEN"

// ldapGroupEnvVar names a real directory-service group the tenant's LDAP /
// cloud-IdP actually has. directory_service_user_group_names is server-matched
// against real groups ("Problem matching limitation user group" 409 otherwise),
// so this path can only be exercised against a tenant with a known group.
const ldapGroupEnvVar = "JAMFPLATFORM_ACC_ENROLLMENT_GROUP_NAME"

// testAccCheckMacAppDestroy verifies apps created during the test were destroyed.
func testAccCheckMacAppDestroy(t *testing.T) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		c := proclassic.New(testhelpers.NewAcceptanceClient(t))
		ctx := context.Background()

		for _, rs := range s.RootModule().Resources {
			if rs.Type != "jamfplatform_pro_mac_app_store_app" {
				continue
			}
			_, err := c.GetMacApplicationByID(ctx, rs.Primary.ID)
			if err != nil {
				if helpers.IsNotFoundError(err) {
					continue
				}
				return fmt.Errorf("error checking Jamf Pro Mac App Store app %s: %s", rs.Primary.ID, err)
			}
			return fmt.Errorf("Jamf Pro Mac App Store app %s still exists", rs.Primary.ID)
		}
		return nil
	}
}

// macAppGeneralOnlyConfig is the import-stable shape: only the required general
// fields. The importer populates general post-Read but leaves optional blocks
// null, so ImportStateVerify must run against a general-only config.
func macAppGeneralOnlyConfig(name, version, deploymentType string) string {
	return fmt.Sprintf(`
		resource "jamfplatform_pro_mac_app_store_app" "test" {
			general = {
				name            = %q
				version         = %q
				bundle_id       = "com.example.tfacc.macapp"
				url             = "https://apps.apple.com/app/id000000001"
				deployment_type = %q
			}
		}
	`, name, version, deploymentType)
}

// macAppFullConfig adds self_service and an all_computers scope on top of general.
func macAppFullConfig(name, version, buttonText string) string {
	return fmt.Sprintf(`
		resource "jamfplatform_pro_mac_app_store_app" "test" {
			general = {
				name            = %q
				version         = %q
				bundle_id       = "com.example.tfacc.macapp"
				url             = "https://apps.apple.com/app/id000000001"
				deployment_type = "Make Available in Self Service"
			}
			scope = {
				all_computers = true
			}
			self_service = {
				install_button_text            = %q
				self_service_description        = "Managed by Terraform acceptance test."
				force_users_to_view_description = false
				feature_on_main_page            = true
			}
		}
	`, name, version, buttonText)
}

// TestAccResource_ProMacApp_Basic exercises create, in-place update, and import
// for the general-only shape. The version/deployment_type change verifies the
// GET-after-Update path (classic UpdateMacApplicationByID returns 201 empty).
func TestAccResource_ProMacApp_Basic(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-pro-macapp-" + suffix

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckMacAppDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: macAppGeneralOnlyConfig(name, "1.0", "Make Available in Self Service"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(macAppResourceAddr, "id"),
					resource.TestCheckResourceAttr(macAppResourceAddr, "general.name", name),
					resource.TestCheckResourceAttr(macAppResourceAddr, "general.version", "1.0"),
					resource.TestCheckResourceAttr(macAppResourceAddr, "general.deployment_type", "Make Available in Self Service"),
				),
			},
			{
				ResourceName:      macAppResourceAddr,
				ImportState:       true,
				ImportStateVerify: true,
				// Timeouts are not returned by the API; ignore on import.
				ImportStateVerifyIgnore: []string{"timeouts"},
			},
			{
				// In-place update: bump version + flip the install method.
				Config: macAppGeneralOnlyConfig(name, "2.0", "Install Automatically/Prompt Users to Install"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(macAppResourceAddr, "general.version", "2.0"),
					resource.TestCheckResourceAttr(macAppResourceAddr, "general.deployment_type", "Install Automatically/Prompt Users to Install"),
				),
			},
		},
	})
}

// TestAccResource_ProMacApp_ScopeAndSelfService exercises the scope + self_service
// blocks and an in-place self_service mutation (not a block deletion — the
// classic PUT is partial-merge and will not clear an omitted block).
func TestAccResource_ProMacApp_ScopeAndSelfService(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-pro-macapp-ss-" + suffix

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckMacAppDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: macAppFullConfig(name, "1.0", "Install"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(macAppResourceAddr, "id"),
					resource.TestCheckResourceAttr(macAppResourceAddr, "scope.all_computers", "true"),
					resource.TestCheckResourceAttr(macAppResourceAddr, "self_service.install_button_text", "Install"),
					resource.TestCheckResourceAttr(macAppResourceAddr, "self_service.feature_on_main_page", "true"),
				),
			},
			{
				Config: macAppFullConfig(name, "1.0", "Get"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(macAppResourceAddr, "self_service.install_button_text", "Get"),
				),
			},
		},
	})
}

// TestAccResource_ProMacApp_InvalidDeploymentType asserts the deployment_type
// OneOf validator rejects an out-of-set value at plan time.
func TestAccResource_ProMacApp_InvalidDeploymentType(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-pro-macapp-bad-dt-" + suffix

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      macAppGeneralOnlyConfig(name, "1.0", "Force Install Right Now"),
				PlanOnly:    true,
				ExpectError: regexp.MustCompile("value must be one of"),
			},
		},
	})
}

// TestAccResource_ProMacApp_AllComputersConflict asserts the shared scope
// validator rejects all_computers = true alongside a computer target.
func TestAccResource_ProMacApp_AllComputersConflict(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-pro-macapp-allc-" + suffix
	config := fmt.Sprintf(`
		resource "jamfplatform_pro_mac_app_store_app" "test" {
			general = {
				name      = %q
				version   = "1.0"
				bundle_id = "com.example.tfacc.macapp"
				url       = "https://apps.apple.com/app/id000000001"
			}
			scope = {
				all_computers = true
				computer_ids  = ["1"]
			}
		}
	`, name)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      config,
				PlanOnly:    true,
				ExpectError: regexp.MustCompile("Conflicts with all-flag"),
			},
		},
	})
}

// TestAccResource_ProMacApp_AllJssUsersConflict asserts the shared scope
// validator rejects all_jss_users = true alongside a user target.
func TestAccResource_ProMacApp_AllJssUsersConflict(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-pro-macapp-allu-" + suffix
	config := fmt.Sprintf(`
		resource "jamfplatform_pro_mac_app_store_app" "test" {
			general = {
				name      = %q
				version   = "1.0"
				bundle_id = "com.example.tfacc.macapp"
				url       = "https://apps.apple.com/app/id000000001"
			}
			scope = {
				all_jss_users = true
				user_ids      = ["1"]
			}
		}
	`, name)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      config,
				PlanOnly:    true,
				ExpectError: regexp.MustCompile("Conflicts with all-flag"),
			},
		},
	})
}

// macAppCategoryFlipConfig builds an app whose category points at one of two
// jamfplatform_pro_category fixtures (selectedCat is "one" or "two").
func macAppCategoryFlipConfig(name, suffix, selectedCat string) string {
	return fmt.Sprintf(`
		resource "jamfplatform_pro_category" "one" {
			name     = "tf-acc-macapp-cat1-%[2]s"
			priority = 9
		}

		resource "jamfplatform_pro_category" "two" {
			name     = "tf-acc-macapp-cat2-%[2]s"
			priority = 9
		}

		resource "jamfplatform_pro_mac_app_store_app" "test" {
			general = {
				name        = %[1]q
				version     = "1.0"
				bundle_id   = "com.example.tfacc.macapp"
				url         = "https://apps.apple.com/app/id000000001"
				category_id = jamfplatform_pro_category.%[3]s.id
			}
		}
	`, name, suffix, selectedCat)
}

// TestAccResource_ProMacApp_CategoryFlip flips category_id between two category
// fixtures. category_name is server-derived from category_id; the second step
// asserts it re-resolves to the new category, guarding against the derived-name
// plan-modifier regression (UseStateForUnknown would pin the stale name and
// trip "produced inconsistent result after apply").
func TestAccResource_ProMacApp_CategoryFlip(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-pro-macapp-cat-" + suffix

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckMacAppDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: macAppCategoryFlipConfig(name, suffix, "one"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(macAppResourceAddr, "general.category_id", "jamfplatform_pro_category.one", "id"),
					resource.TestCheckResourceAttr(macAppResourceAddr, "general.category_name", "tf-acc-macapp-cat1-"+suffix),
				),
			},
			{
				Config: macAppCategoryFlipConfig(name, suffix, "two"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(macAppResourceAddr, "general.category_id", "jamfplatform_pro_category.two", "id"),
					resource.TestCheckResourceAttr(macAppResourceAddr, "general.category_name", "tf-acc-macapp-cat2-"+suffix),
				),
			},
		},
	})
}

// Jamf Parent App Store metadata, captured from the iTunes Lookup API
// (https://itunes.apple.com/lookup?id=1458797105) on 2026-05-31. These are
// stored verbatim by /macapplications (no App Store resolution), so they are
// pinned here rather than fetched at runtime to keep the test deterministic.
// The bundle_id is what links the created app to the VPP location's synced
// content for device-based assignment.
const (
	jamfParentName     = "Jamf Parent"
	jamfParentVersion  = "5.2.4"
	jamfParentBundleID = "com.jamf.parent"
	jamfParentURL      = "https://apps.apple.com/us/app/jamf-parent/id1458797105?uo=4"
	jamfParentDesc     = `Jamf Parent empowers parents to manage their children's school-issued devices. Using the intuitive interface, you can restrict which apps your child can access on their device, receive notifications when your child arrives at school, and schedule homework time or bedtime by using Device Rules to allow or restrict certain apps.

Key features:


- Restrict and allow apps in real time (including games and social media)
- Restrict and allow device features (including the camera)
- Restrict and allow websites
- Create scheduled app restrictions for homework time, bedtime, and timeout`
	jamfParentGenre = "Education"
	// artworkUrl512 from the same iTunes lookup — uploaded via the
	// jamfplatform_pro_icon resource and referenced by id, so the icon is
	// embedded through the proper composition rather than inline upload.
	jamfParentArtworkURL = "https://is1-ssl.mzstatic.com/image/thumb/Purple221/v4/62/6e/bb/626ebb84-337a-9546-1b80-77843f3ad105/AppIcon-0-0-1x_U007epad-0-9-0-0-sRGB-GLES2_U002c0-85-220.png/512x512bb.jpg"
)

// macAppVPPFullMetadataConfig builds a self-contained config: a VPP location
// created from the env-supplied ABM token, a category, and a Mac App Store app
// populated with the full Jamf Parent metadata and assigned device-based VPP
// licenses from that location. The app's name is suffixed for uniqueness; VPP
// content matching keys on bundle_id, which is pinned to the real value.
func macAppVPPFullMetadataConfig(token, suffix, buttonText string) string {
	return fmt.Sprintf(`
		resource "jamfplatform_pro_volume_purchasing_location" "vpp" {
			name                                     = "tf-acc-macapp-vpp-%[2]s"
			service_token                            = %[1]q
			service_token_wo_version                 = 1
			automatically_populate_purchased_content = true
		}

		resource "jamfplatform_pro_category" "edu" {
			name     = "tf-acc-macapp-edu-%[2]s"
			priority = 9
		}

		resource "jamfplatform_pro_icon" "jamf_parent" {
			icon_file_source = %[9]q
		}

		resource "jamfplatform_pro_mac_app_store_app" "test" {
			general = {
				name            = "%[3]s tf-acc %[2]s"
				version         = %[4]q
				bundle_id       = %[5]q
				url             = %[6]q
				is_free         = true
				deployment_type = "Make Available in Self Service"
				category_id     = jamfplatform_pro_category.edu.id
			}

			scope = {
				all_computers = true
			}

			self_service = {
				install_button_text             = %[8]q
				self_service_description         = %[7]q
				force_users_to_view_description  = false
				feature_on_main_page             = true
				notification_enabled             = true
				notification_method              = "Self Service"
				notification_subject             = "Jamf Parent is available"
				notification_message             = "Install Jamf Parent from Self Service."

				self_service_icon = {
					id = jamfplatform_pro_icon.jamf_parent.id
				}

				self_service_categories = [
					{
						id         = jamfplatform_pro_category.edu.id
						display_in = true
						feature_in = false
					},
				]
			}

			vpp = {
				assign_vpp_device_based_licenses = true
				vpp_admin_account_id             = jamfplatform_pro_volume_purchasing_location.vpp.id
			}
		}
	`, token, suffix, jamfParentName, jamfParentVersion, jamfParentBundleID, jamfParentURL, jamfParentDesc, buttonText, jamfParentArtworkURL)
}

// TestAccResource_ProMacApp_VPPFullMetadata is the full-coverage gated test: it
// exercises every embeddable field at once — full general metadata (incl.
// is_free + category), an all_computers scope, the complete self_service block
// (description, notification, the SetNested self_service_categories diff-by-id
// path, and a self_service_icon referencing a jamfplatform_pro_icon resource
// that uploads Jamf Parent's real App Store artwork), and the top-level vpp
// block with device-based assignment.
//
// Gated on JAMFPLATFORM_VPP_TOKEN: device-based VPP assignment 409s ("App is
// not available for device assignment") unless the token's location owns
// device-assignable licenses for bundle_id com.jamf.parent. The in-place update
// flips install_button_text and assign_vpp_device_based_licenses true->false
// (wire-probed to round-trip), verifying the GET-after-Update path on the vpp
// block. The icon is embedded via composition (jamfplatform_pro_icon handles
// upload + the server-side PNG re-encode); mac_app references it by id, so no
// inline icon upload is needed.
func TestAccResource_ProMacApp_VPPFullMetadata(t *testing.T) {
	token := os.Getenv(vppTokenEnvVar)
	if token == "" {
		t.Skipf("%s not set; skipping full-metadata VPP acceptance test", vppTokenEnvVar)
	}
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckMacAppDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: macAppVPPFullMetadataConfig(token, suffix, "Get"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(macAppResourceAddr, "id"),
					resource.TestCheckResourceAttr(macAppResourceAddr, "general.version", jamfParentVersion),
					resource.TestCheckResourceAttr(macAppResourceAddr, "general.bundle_id", jamfParentBundleID),
					resource.TestCheckResourceAttr(macAppResourceAddr, "general.url", jamfParentURL),
					resource.TestCheckResourceAttr(macAppResourceAddr, "general.is_free", "true"),
					resource.TestCheckResourceAttrSet(macAppResourceAddr, "general.category_id"),
					resource.TestCheckResourceAttr(macAppResourceAddr, "scope.all_computers", "true"),
					resource.TestCheckResourceAttr(macAppResourceAddr, "self_service.install_button_text", "Get"),
					resource.TestCheckResourceAttr(macAppResourceAddr, "self_service.self_service_description", jamfParentDesc),
					resource.TestCheckResourceAttr(macAppResourceAddr, "self_service.notification_enabled", "true"),
					resource.TestCheckResourceAttr(macAppResourceAddr, "self_service.self_service_categories.#", "1"),
					resource.TestCheckResourceAttrPair(macAppResourceAddr, "self_service.self_service_icon.id", "jamfplatform_pro_icon.jamf_parent", "id"),
					resource.TestCheckResourceAttrSet(macAppResourceAddr, "self_service.self_service_icon.uri"),
					resource.TestCheckResourceAttr(macAppResourceAddr, "vpp.assign_vpp_device_based_licenses", "true"),
					resource.TestCheckResourceAttrSet(macAppResourceAddr, "vpp.vpp_admin_account_id"),
					resource.TestCheckResourceAttrSet(macAppResourceAddr, "vpp.total_vpp_licenses"),
				),
			},
			{
				Config: strings.Replace(
					macAppVPPFullMetadataConfig(token, suffix, "Install"),
					"assign_vpp_device_based_licenses = true",
					"assign_vpp_device_based_licenses = false",
					1,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(macAppResourceAddr, "self_service.install_button_text", "Install"),
					resource.TestCheckResourceAttr(macAppResourceAddr, "vpp.assign_vpp_device_based_licenses", "false"),
				),
			},
		},
	})
}

// macAppScopeTargetsConfig builds sibling building/department/network_segment
// resources and references their IDs from the app's scope — exercising the
// scope target/limitation/exclusion ID round-trips that the all_computers
// happy-paths never touch. directoryNames is interpolated into the
// directory-service name sets (plain strings, no sibling needed). The two
// computer/user target categories are omitted because they need real tenant
// inventory; everything else is hermetic.
func macAppScopeTargetsConfig(suffix string, limitationUserNames string) string {
	return fmt.Sprintf(`
		resource "jamfplatform_pro_building" "b1" {
			name = "tf-acc-macapp-bldg-%[1]s"
		}

		resource "jamfplatform_pro_department" "d1" {
			name = "tf-acc-macapp-dept-%[1]s"
		}

		resource "jamfplatform_pro_network_segment" "n1" {
			name             = "tf-acc-macapp-ns-%[1]s"
			starting_address = "10.123.0.1"
			ending_address   = "10.123.0.254"
		}

		resource "jamfplatform_pro_mac_app_store_app" "test" {
			general = {
				name      = "tf-acc-pro-macapp-scope-%[1]s"
				version   = "1.0"
				bundle_id = "com.example.tfacc.macapp.scope"
				url       = "https://apps.apple.com/app/id000000002"
			}

			scope = {
				building_ids   = [jamfplatform_pro_building.b1.id]
				department_ids = [jamfplatform_pro_department.d1.id]

				limitations = {
					network_segment_ids                   = [jamfplatform_pro_network_segment.n1.id]
					directory_service_or_local_user_names  = %[2]s
					# directory_service_user_group_names omitted: the server matches
					# it against real LDAP groups ("Problem matching limitation user
					# group" 409), so it can't be exercised hermetically.
				}

				exclusions = {
					network_segment_ids                    = [jamfplatform_pro_network_segment.n1.id]
					directory_service_or_local_user_names   = ["excluded-user"]
				}
			}
		}
	`, suffix, limitationUserNames)
}

// TestAccResource_ProMacApp_ScopeTargets exercises the scope target / limitation
// / exclusion ID round-trips against real sibling resources, and (step 2) a
// set-shrink: the limitation directory-user name set goes from two entries to
// one, verifying the nested set updates cleanly under the partial-merge PUT.
func TestAccResource_ProMacApp_ScopeTargets(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckMacAppDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: macAppScopeTargetsConfig(suffix, `["alice", "bob"]`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(macAppResourceAddr, "id"),
					resource.TestCheckResourceAttr(macAppResourceAddr, "scope.building_ids.#", "1"),
					resource.TestCheckResourceAttr(macAppResourceAddr, "scope.department_ids.#", "1"),
					resource.TestCheckResourceAttrPair(macAppResourceAddr, "scope.building_ids.0", "jamfplatform_pro_building.b1", "id"),
					resource.TestCheckResourceAttr(macAppResourceAddr, "scope.limitations.network_segment_ids.#", "1"),
					resource.TestCheckResourceAttr(macAppResourceAddr, "scope.limitations.directory_service_or_local_user_names.#", "2"),
					resource.TestCheckResourceAttr(macAppResourceAddr, "scope.exclusions.network_segment_ids.#", "1"),
					resource.TestCheckResourceAttr(macAppResourceAddr, "scope.exclusions.directory_service_or_local_user_names.#", "1"),
				),
			},
			{
				// Set-shrink: drop one directory-user name (2 -> 1).
				Config: macAppScopeTargetsConfig(suffix, `["alice"]`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(macAppResourceAddr, "scope.limitations.directory_service_or_local_user_names.#", "1"),
				),
			},
		},
	})
}

// TestAccDataSource_ProMacApp resolves a created app through the singular data
// source by ID and by name, asserting the flat read-only projection.
func TestAccDataSource_ProMacApp(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-pro-macapp-ds-" + suffix

	config := fmt.Sprintf(`
		resource "jamfplatform_pro_mac_app_store_app" "test" {
			general = {
				name      = %q
				version   = "3.1"
				bundle_id = "com.example.tfacc.macapp.ds"
				url       = "https://apps.apple.com/app/id000000003"
				is_free   = true
			}
		}

		data "jamfplatform_pro_mac_app_store_app" "by_id" {
			id = jamfplatform_pro_mac_app_store_app.test.id
		}

		data "jamfplatform_pro_mac_app_store_app" "by_name" {
			name = jamfplatform_pro_mac_app_store_app.test.general.name
		}
	`, name)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckMacAppDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.jamfplatform_pro_mac_app_store_app.by_id", "name", macAppResourceAddr, "general.name"),
					resource.TestCheckResourceAttr("data.jamfplatform_pro_mac_app_store_app.by_id", "version", "3.1"),
					resource.TestCheckResourceAttr("data.jamfplatform_pro_mac_app_store_app.by_id", "bundle_id", "com.example.tfacc.macapp.ds"),
					resource.TestCheckResourceAttr("data.jamfplatform_pro_mac_app_store_app.by_id", "is_free", "true"),
					resource.TestCheckResourceAttrPair("data.jamfplatform_pro_mac_app_store_app.by_name", "id", macAppResourceAddr, "id"),
					resource.TestCheckResourceAttr("data.jamfplatform_pro_mac_app_store_app.by_name", "name", name),
				),
			},
		},
	})
}

// TestAccListResource_ProMacApp exercises the list resource via the
// `terraform query` workflow, filtering to the unique created app by name.
// The list is identity-only, so the check asserts the filtered result count.
func TestAccListResource_ProMacApp(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-pro-macapp-list-" + suffix

	resource.Test(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_14_0),
		},
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckMacAppDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "jamfplatform_pro_mac_app_store_app" "test" {
						general = {
							name      = %q
							version   = "1.0"
							bundle_id = "com.example.tfacc.macapp.list"
							url       = "https://apps.apple.com/app/id000000004"
						}
					}
				`, name),
				Check: resource.TestCheckResourceAttrSet(macAppResourceAddr, "id"),
			},
			{
				Query: true,
				Config: fmt.Sprintf(`
					provider "jamfplatform" {}

					list "jamfplatform_pro_mac_app_store_app" "test" {
						provider = jamfplatform

						config {
							filter = {
								name_substring = %q
							}
						}
					}
				`, name),
				// The list is identity-only (no include_resource), so assert the
				// filtered result count rather than per-resource attribute values.
				QueryResultChecks: []querycheck.QueryResultCheck{
					querycheck.ExpectLength("jamfplatform_pro_mac_app_store_app.test", 1),
				},
			},
		},
	})
}

// TestAccResource_ProMacApp_ScopeLdapGroup exercises a limitation that
// references a real directory-service user group, which the server validates
// against the configured LDAP / cloud-IdP. Gated on
// JAMFPLATFORM_ACC_ENROLLMENT_GROUP_NAME (a group the tenant actually has);
// also serves as the live check that the plan-time DS-group preflight accepts a
// real group rather than rejecting it.
func TestAccResource_ProMacApp_ScopeLdapGroup(t *testing.T) {
	group := os.Getenv(ldapGroupEnvVar)
	if group == "" {
		t.Skipf("%s not set; skipping directory-service group scope test", ldapGroupEnvVar)
	}
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-pro-macapp-ldap-" + suffix

	config := fmt.Sprintf(`
		resource "jamfplatform_pro_mac_app_store_app" "test" {
			general = {
				name      = %q
				version   = "1.0"
				bundle_id = "com.example.tfacc.macapp.ldap"
				url       = "https://apps.apple.com/app/id000000005"
			}

			scope = {
				all_computers = true

				limitations = {
					directory_service_user_group_names = [%q]
				}
			}
		}
	`, name, group)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckMacAppDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(macAppResourceAddr, "id"),
					resource.TestCheckResourceAttr(macAppResourceAddr, "scope.limitations.directory_service_user_group_names.#", "1"),
					resource.TestCheckResourceAttr(macAppResourceAddr, "scope.limitations.directory_service_user_group_names.0", group),
				),
			},
		},
	})
}
