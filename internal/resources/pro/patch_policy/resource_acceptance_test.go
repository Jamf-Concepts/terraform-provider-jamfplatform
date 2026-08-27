// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

// Tests in this file talk to the Jamf ProClassic /patchpolicies endpoint.
// Classic has known concurrency issues when multiple writes hit the same
// resource type — keep these tests serial with any future classic acceptance
// work in this package.
//
// FIXTURE CHAIN (mandatory): a patch policy can only target a version that has a
// package assigned on its title. Each test therefore stands up:
//   jamfplatform_pro_package          (metadata-only)
//     → jamfplatform_pro_patch_software_title (name_id "285"/source_id 1, with
//        version_packages = { "8.33.2.2" = <package id> })
//        → jamfplatform_pro_patch_policy (software_title_configuration_id =
//           <title id>, target_version = "8.33.2.2").
//
// The fixtures use the real "8x8 Work" patch catalog entry: name_id "285",
// source_id 1, with version "8.33.2.2". If the catalog entry or that version is
// removed from the test tenant's patch source, these tests will need updating.

package patch_policy_test

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/proclassic"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/querycheck"
	"github.com/hashicorp/terraform-plugin-testing/querycheck/queryfilter"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/testhelpers"
)

const (
	accTitleNameID   = "285" // 8x8 Work
	accTitleSourceID = 1
	accTitleVersion  = "8.33.2.2"
	patchPolicyType  = "jamfplatform_pro_patch_policy"
)

// testAccCheckPatchPolicyDestroy verifies patch policies created during the test
// were destroyed.
func testAccCheckPatchPolicyDestroy(t *testing.T) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		c := proclassic.New(testhelpers.NewAcceptanceClient(t))
		ctx := context.Background()

		for _, rs := range s.RootModule().Resources {
			if rs.Type != patchPolicyType {
				continue
			}
			_, err := c.GetPatchPolicyByID(ctx, rs.Primary.ID)
			if err != nil {
				if helpers.IsNotFoundError(err) {
					continue
				}
				return fmt.Errorf("error checking Jamf Pro patch policy %s: %s", rs.Primary.ID, err)
			}
			return fmt.Errorf("Jamf Pro patch policy %s still exists", rs.Primary.ID)
		}
		return nil
	}
}

// fixtureTitle returns the package + patch software title fixture HCL shared by
// every step. The title assigns package A to the target version so the policy
// can target it.
func fixtureTitle(suffix string) string {
	return fmt.Sprintf(`
		resource "jamfplatform_pro_package" "pkg_a" {
			display_name = "tf-acc-pp-pkgA-%[1]s"
			file_name    = "tf-acc-pp-pkgA-%[1]s.pkg"
		}

		resource "jamfplatform_pro_patch_software_title" "title" {
			name      = "tf-acc 8x8 Work pp %[1]s"
			name_id   = %[2]q
			source_id = %[3]d

			version_packages = {
				%[4]q = jamfplatform_pro_package.pkg_a.id
			}
		}
	`, suffix, accTitleNameID, accTitleSourceID, accTitleVersion)
}

// TestAccResource_ProPatchPolicy_Basic exercises create, then a multi-attribute
// in-place update mutating every non-RequiresReplace writable attribute:
// enabled, distribution_method (selfservice→prompt), allow_downgrade,
// patch_unknown, scope (add a computer_group / building, then clear the
// building with a declared `[]`), and several user_interaction fields
// (reminders / deadline / grace). It asserts the computed server-derived
// fields populate throughout. Finally import with justified ignores.
func TestAccResource_ProPatchPolicy_Basic(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	bld := "tf-acc-pp-bld-" + suffix
	grp := "tf-acc-pp-grp-" + suffix
	// computer_group_ids uses a self-minted smart computer group via
	// jamfplatform_device_group (whose jamf_pro_id is the classic group ID the
	// scope wants). The title fixture sets no site, so a siteless Full-JSS smart
	// group is accepted and the policy can run enabled=true. The building is a
	// real fixture.

	// Step 1: create as Self Service with a minimal user_interaction block.
	stepCreate := fixtureTitle(suffix) + fmt.Sprintf(`
		resource "jamfplatform_pro_patch_policy" "test" {
			software_title_configuration_id = jamfplatform_pro_patch_software_title.title.id
			name                            = "tf-acc-pp-%[1]s"
			target_version                  = %[2]q
			enabled                         = false
			distribution_method             = "selfservice"
			allow_downgrade                 = false
			patch_unknown                   = false

			user_interaction = {
				install_button_text = "Update"

				deadlines = {
					enabled = true
					period  = 7
				}
			}
		}
	`, suffix, accTitleVersion)

	// Step 2: flip every writable general bool/enum, add scope (the default smart
	// group + a building), and mutate user_interaction reminders/deadline/grace.
	stepUpdate := fixtureTitle(suffix) + fmt.Sprintf(`
		resource "jamfplatform_pro_building" "bld" {
			name = %[3]q
		}

		resource "jamfplatform_device_group" "grp" {
			name        = %[4]q
			group_type  = "smart"
			device_type = "computer"
			description = "Patch policy acceptance fixture"
			criteria = [
				{ criteria = "Operating System Version", operator = "greater than or equal", value = "10.0" },
			]
		}

		resource "jamfplatform_pro_patch_policy" "test" {
			software_title_configuration_id = jamfplatform_pro_patch_software_title.title.id
			name                            = "tf-acc-pp-%[1]s"
			target_version                  = %[2]q
			enabled                         = true
			distribution_method             = "prompt"
			allow_downgrade                 = true
			patch_unknown                   = true

			scope = {
				targets = {
					computer_group_ids = [jamfplatform_device_group.grp.jamf_pro_id]
					building_ids       = [jamfplatform_pro_building.bld.id]
				}
			}

			user_interaction = {
				install_button_text      = "Install Now"
				self_service_description = "Updated description"

				notifications = {
					enabled = true
					subject = "Update available"
					reminders = {
						enabled   = true
						frequency = 12
					}
				}

				deadlines = {
					enabled = true
					period  = 3
				}

				grace_period = {
					duration                    = 30
					notification_center_subject = "Heads up"
				}
			}
		}
	`, suffix, accTitleVersion, bld, grp)

	// Step 3: clear the building target with an explicit `[]` (keep the group)
	// to exercise the declared-clear path. Under granular scope ownership,
	// omitting building_ids instead would drop it from state and leave the
	// category to the admin UI (preserved by the read-merge-write update), not
	// clear it.
	stepScopeShrink := fixtureTitle(suffix) + fmt.Sprintf(`
		resource "jamfplatform_pro_building" "bld" {
			name = %[3]q
		}

		resource "jamfplatform_device_group" "grp" {
			name        = %[4]q
			group_type  = "smart"
			device_type = "computer"
			description = "Patch policy acceptance fixture"
			criteria = [
				{ criteria = "Operating System Version", operator = "greater than or equal", value = "10.0" },
			]
		}

		resource "jamfplatform_pro_patch_policy" "test" {
			software_title_configuration_id = jamfplatform_pro_patch_software_title.title.id
			name                            = "tf-acc-pp-%[1]s"
			target_version                  = %[2]q
			enabled                         = true
			distribution_method             = "prompt"
			allow_downgrade                 = true
			patch_unknown                   = true

			scope = {
				targets = {
					computer_group_ids = [jamfplatform_device_group.grp.jamf_pro_id]
					building_ids       = []
				}
			}

			user_interaction = {
				install_button_text = "Install Now"

				deadlines = {
					enabled = true
					period  = 3
				}
			}
		}
	`, suffix, accTitleVersion, bld, grp)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPatchPolicyDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: stepCreate,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(patchPolicyType+".test", "id"),
					resource.TestCheckResourceAttrPair(patchPolicyType+".test", "software_title_configuration_id", "jamfplatform_pro_patch_software_title.title", "id"),
					resource.TestCheckResourceAttr(patchPolicyType+".test", "target_version", accTitleVersion),
					resource.TestCheckResourceAttr(patchPolicyType+".test", "distribution_method", "selfservice"),
					resource.TestCheckResourceAttr(patchPolicyType+".test", "allow_downgrade", "false"),
					resource.TestCheckResourceAttr(patchPolicyType+".test", "patch_unknown", "false"),
					// Server-derived fields populate from the patch definition.
					resource.TestCheckResourceAttrSet(patchPolicyType+".test", "release_date"),
					resource.TestCheckResourceAttrSet(patchPolicyType+".test", "incremental_update"),
					resource.TestCheckResourceAttrSet(patchPolicyType+".test", "reboot"),
					resource.TestCheckResourceAttrSet(patchPolicyType+".test", "minimum_os"),
					resource.TestCheckResourceAttrSet(patchPolicyType+".test", "kill_apps.#"),
				),
			},
			{
				Config: stepUpdate,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(patchPolicyType+".test", "enabled", "true"),
					resource.TestCheckResourceAttrPair(patchPolicyType+".test", "scope.targets.computer_group_ids.0", "jamfplatform_device_group.grp", "jamf_pro_id"),
					resource.TestCheckResourceAttr(patchPolicyType+".test", "distribution_method", "prompt"),
					resource.TestCheckResourceAttr(patchPolicyType+".test", "allow_downgrade", "true"),
					resource.TestCheckResourceAttr(patchPolicyType+".test", "patch_unknown", "true"),
					resource.TestCheckResourceAttr(patchPolicyType+".test", "scope.targets.computer_group_ids.#", "1"),
					resource.TestCheckResourceAttr(patchPolicyType+".test", "scope.targets.building_ids.#", "1"),
					resource.TestCheckResourceAttr(patchPolicyType+".test", "user_interaction.install_button_text", "Install Now"),
					resource.TestCheckResourceAttr(patchPolicyType+".test", "user_interaction.notifications.reminders.frequency", "12"),
					resource.TestCheckResourceAttr(patchPolicyType+".test", "user_interaction.deadlines.period", "3"),
					resource.TestCheckResourceAttr(patchPolicyType+".test", "user_interaction.grace_period.duration", "30"),
					// Server-derived still present.
					resource.TestCheckResourceAttrSet(patchPolicyType+".test", "kill_apps.#"),
				),
			},
			{
				Config: stepScopeShrink,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(patchPolicyType+".test", "scope.targets.computer_group_ids.#", "1"),
					// The declared `[]` clear round-trips as an empty set — the
					// canonical "no members" value for a managed category.
					resource.TestCheckResourceAttr(patchPolicyType+".test", "scope.targets.building_ids.#", "0"),
				),
			},
			{
				ResourceName:      patchPolicyType + ".test",
				ImportState:       true,
				ImportStateVerify: true,
				// timeouts: framework-only, never round-trips.
				// scope: import hydrates every category; apply keeps
				// declared-only, so the hydrated import state legitimately
				// differs from this subset-scope config.
				// user_interaction: Optional state-gated block — import hydrates
				// it fully while this config declares a subset.
				// software_title_configuration_id: GET does NOT echo it (it is a
				// create-time-only path parameter, wire-probed); it cannot be
				// reconstructed on import.
				ImportStateVerifyIgnore: []string{"timeouts", "scope", "user_interaction", "software_title_configuration_id"},
			},
		},
	})
}

// TestAccResource_ProPatchPolicy_InvalidDistributionMethod asserts the
// distribution_method OneOf validator rejects an unknown value at plan time. The
// error detail wraps at ~80 cols; the regex anchors on the no-space token
// "selfservice" to avoid a whitespace wrap point.
func TestAccResource_ProPatchPolicy_InvalidDistributionMethod(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fixtureTitle(suffix) + fmt.Sprintf(`
					resource "jamfplatform_pro_patch_policy" "test" {
						software_title_configuration_id = jamfplatform_pro_patch_software_title.title.id
						name                            = "tf-acc-pp-invalid-%[1]s"
						target_version                  = %[2]q
						distribution_method             = "notarealmethod"
					}
				`, suffix, accTitleVersion),
				ExpectError: regexp.MustCompile(`selfservice`),
			},
		},
	})
}

// TestAccDataSource_ProPatchPolicy_ByID looks up a freshly-created patch policy
// by ID.
func TestAccDataSource_ProPatchPolicy_ByID(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPatchPolicyDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: fixtureTitle(suffix) + fmt.Sprintf(`
					resource "jamfplatform_pro_patch_policy" "src" {
						software_title_configuration_id = jamfplatform_pro_patch_software_title.title.id
						name                            = "tf-acc-pp-ds-%[1]s"
						target_version                  = %[2]q
					}

					data "jamfplatform_pro_patch_policy" "lookup" {
						id = jamfplatform_pro_patch_policy.src.id
					}
				`, suffix, accTitleVersion),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.jamfplatform_pro_patch_policy.lookup", "name", "jamfplatform_pro_patch_policy.src", "name"),
					resource.TestCheckResourceAttr("data.jamfplatform_pro_patch_policy.lookup", "target_version", accTitleVersion),
					resource.TestCheckResourceAttrSet("data.jamfplatform_pro_patch_policy.lookup", "kill_apps.#"),
				),
			},
		},
	})
}

// TestAccListResource_ProPatchPolicy_Basic exercises the list resource via the
// `terraform query` workflow. The classic list endpoint surfaces the policy
// display name, so DisplayName / the filter match on the name.
func TestAccListResource_ProPatchPolicy_Basic(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-pp-list-" + suffix

	resource.Test(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_14_0),
		},
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPatchPolicyDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: fixtureTitle(suffix) + fmt.Sprintf(`
					resource "jamfplatform_pro_patch_policy" "src" {
						software_title_configuration_id = jamfplatform_pro_patch_software_title.title.id
						name                            = %[1]q
						target_version                  = %[2]q
					}
				`, name, accTitleVersion),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("jamfplatform_pro_patch_policy.src", "id"),
				),
			},
			{
				Query: true,
				Config: fmt.Sprintf(`
					provider "jamfplatform" {}

					list "jamfplatform_pro_patch_policy" "test" {
						provider         = jamfplatform
						include_resource = true

						config {
							filter = {
								name_substring = %q
							}
						}
					}
				`, name),
				QueryResultChecks: []querycheck.QueryResultCheck{
					querycheck.ExpectResourceKnownValues(
						"jamfplatform_pro_patch_policy.test",
						queryfilter.ByDisplayName(knownvalue.StringExact(name)),
						[]querycheck.KnownValueCheck{
							{Path: tfjsonpath.New("name"), KnownValue: knownvalue.StringExact(name)},
						},
					),
				},
			},
		},
	})
}
