// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

// Tests in this file talk to the Jamf ProClassic /patchsoftwaretitles endpoint
// (deprecated in the spec but the only functional CRUD surface — see crud.go).
// Classic has known concurrency issues when multiple writes hit the same
// resource type — keep these tests serial with any future classic acceptance
// work in this package.
//
// The fixtures use the real "8x8 Work" patch catalog entry: name_id "285",
// source_id 1, with version "8.33.2.2". If the catalog entry or that version is
// removed from the test tenant's patch source, these tests will need updating.

package patch_software_title_test

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
	accTitleNameID    = "285" // 8x8 Work
	accTitleSourceID  = 1
	accTitleVersion   = "8.33.2.2"
	patchSoftwareType = "jamfplatform_pro_patch_software_title"
)

// testAccCheckPatchSoftwareTitleDestroy verifies titles created during the test
// were destroyed.
func testAccCheckPatchSoftwareTitleDestroy(t *testing.T) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		c := proclassic.New(testhelpers.NewAcceptanceClient(t))
		ctx := context.Background()

		for _, rs := range s.RootModule().Resources {
			if rs.Type != patchSoftwareType {
				continue
			}
			_, err := c.GetPatchSoftwareTitleByID(ctx, rs.Primary.ID)
			if err != nil {
				if helpers.IsNotFoundError(err) {
					continue
				}
				return fmt.Errorf("error checking Jamf Pro patch software title %s: %s", rs.Primary.ID, err)
			}
			return fmt.Errorf("Jamf Pro patch software title %s still exists", rs.Primary.ID)
		}
		return nil
	}
}

// TestAccResource_ProPatchSoftwareTitle_Basic exercises create, then a
// multi-attribute in-place update mutating every non-RequiresReplace attribute:
// category_id, site_id, both notification bools, and version_packages (add a key
// in step 2, then change which package it points at AND remove it in step 3 to
// exercise both the assign and the empty-package clear/unassign path). Finally
// import with justified ignores.
func TestAccResource_ProPatchSoftwareTitle_Basic(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	cat := "tf-acc-pst-cat-" + suffix
	site := "tf-acc-pst-site-" + suffix
	pkgA := "tf-acc-pst-pkgA-" + suffix
	pkgB := "tf-acc-pst-pkgB-" + suffix

	// Step 1: mint the title with defaults; assign nothing.
	stepCreate := fmt.Sprintf(`
		resource "jamfplatform_pro_package" "pkg_a" {
			display_name = %q
			file_name    = "%s.pkg"
		}

		resource "jamfplatform_pro_patch_software_title" "test" {
			name      = "tf-acc 8x8 Work %[3]s"
			name_id   = %q
			source_id = %d
		}
	`, pkgA, pkgA, suffix, accTitleNameID, accTitleSourceID)

	// Step 2: add category + site + flip notifications + assign package A to a
	// real version.
	stepAssign := fmt.Sprintf(`
		resource "jamfplatform_pro_category" "cat" {
			name     = %q
			priority = 9
		}

		resource "jamfplatform_pro_site" "site" {
			name = %q
		}

		resource "jamfplatform_pro_package" "pkg_a" {
			display_name = %q
			file_name    = "%s.pkg"
		}

		resource "jamfplatform_pro_patch_software_title" "test" {
			name               = "tf-acc 8x8 Work %[5]s"
			name_id            = %q
			source_id          = %d
			category_id        = jamfplatform_pro_category.cat.id
			site_id            = jamfplatform_pro_site.site.id
			web_notification   = false
			email_notification = false

			version_packages = {
				%q = jamfplatform_pro_package.pkg_a.id
			}
		}
	`, cat, site, pkgA, pkgA, suffix, accTitleNameID, accTitleSourceID, accTitleVersion)

	// Step 3: point the same version at a different package (change) and drop the
	// previous key by not declaring it for another version (remove → clear). Here
	// we change which package version A points at, exercising re-assign; the
	// clear path is covered by the eventual destroy and by the unit tests, but we
	// also reduce to an empty map to verify clear-on-remove round-trips.
	stepClear := fmt.Sprintf(`
		resource "jamfplatform_pro_category" "cat" {
			name     = %q
			priority = 9
		}

		resource "jamfplatform_pro_site" "site" {
			name = %q
		}

		resource "jamfplatform_pro_package" "pkg_a" {
			display_name = %q
			file_name    = "%s.pkg"
		}

		resource "jamfplatform_pro_package" "pkg_b" {
			display_name = %q
			file_name    = "%s.pkg"
		}

		resource "jamfplatform_pro_patch_software_title" "test" {
			name               = "tf-acc 8x8 Work %[7]s"
			name_id            = %q
			source_id          = %d
			category_id        = "-1"
			site_id            = "-1"
			web_notification   = true
			email_notification = true

			# Re-point version A at package B (change), and the prior key for
			# version A is retained but now clears nothing extra. Removing version
			# A entirely from the map on a later apply would emit the empty-package
			# clear; here we re-assign to prove the assign path mutates in place.
			version_packages = {
				%q = jamfplatform_pro_package.pkg_b.id
			}
		}
	`, cat, site, pkgA, pkgA, pkgB, pkgB, suffix, accTitleNameID, accTitleSourceID, accTitleVersion)

	// Step 4: remove the version_packages key entirely → exercises the
	// empty-package clear/unassign path.
	stepUnassign := fmt.Sprintf(`
		resource "jamfplatform_pro_package" "pkg_a" {
			display_name = %q
			file_name    = "%s.pkg"
		}

		resource "jamfplatform_pro_package" "pkg_b" {
			display_name = %q
			file_name    = "%s.pkg"
		}

		resource "jamfplatform_pro_patch_software_title" "test" {
			name      = "tf-acc 8x8 Work %[5]s"
			name_id   = %q
			source_id = %d
		}
	`, pkgA, pkgA, pkgB, pkgB, suffix, accTitleNameID, accTitleSourceID)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPatchSoftwareTitleDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: stepCreate,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(patchSoftwareType+".test", "id"),
					resource.TestCheckResourceAttr(patchSoftwareType+".test", "name_id", accTitleNameID),
					resource.TestCheckResourceAttr(patchSoftwareType+".test", "source_id", "1"),
					// Server defaults.
					resource.TestCheckResourceAttr(patchSoftwareType+".test", "category_id", "-1"),
					resource.TestCheckResourceAttr(patchSoftwareType+".test", "site_id", "-1"),
					// Catalog versions populated by the server.
					resource.TestCheckResourceAttrSet(patchSoftwareType+".test", "available_versions.#"),
				),
			},
			{
				Config: stepAssign,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(patchSoftwareType+".test", "category_id", "jamfplatform_pro_category.cat", "id"),
					resource.TestCheckResourceAttrPair(patchSoftwareType+".test", "site_id", "jamfplatform_pro_site.site", "id"),
					resource.TestCheckResourceAttr(patchSoftwareType+".test", "web_notification", "false"),
					resource.TestCheckResourceAttr(patchSoftwareType+".test", "email_notification", "false"),
					resource.TestCheckResourceAttrPair(patchSoftwareType+".test", "version_packages."+accTitleVersion, "jamfplatform_pro_package.pkg_a", "id"),
				),
			},
			{
				Config: stepClear,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(patchSoftwareType+".test", "category_id", "-1"),
					resource.TestCheckResourceAttr(patchSoftwareType+".test", "site_id", "-1"),
					resource.TestCheckResourceAttr(patchSoftwareType+".test", "web_notification", "true"),
					resource.TestCheckResourceAttr(patchSoftwareType+".test", "email_notification", "true"),
					// Version A now points at package B.
					resource.TestCheckResourceAttrPair(patchSoftwareType+".test", "version_packages."+accTitleVersion, "jamfplatform_pro_package.pkg_b", "id"),
				),
			},
			{
				Config: stepUnassign,
				Check: resource.ComposeAggregateTestCheckFunc(
					// version_packages fully cleared (empty-package unassign path).
					resource.TestCheckNoResourceAttr(patchSoftwareType+".test", "version_packages.%"),
				),
			},
			{
				ResourceName:      patchSoftwareType + ".test",
				ImportState:       true,
				ImportStateVerify: true,
				// timeouts: framework-only, never round-trips.
				// version_packages: managed-subset map keyed off prior state; on
				// import there is no prior state, so it cannot be reconstructed.
				// available_versions / *_name: server-derived and at this step the
				// state already matches, but version_packages is the import gap.
				ImportStateVerifyIgnore: []string{"timeouts", "version_packages"},
			},
		},
	})
}

// TestAccResource_ProPatchSoftwareTitle_InvalidPackageID asserts the
// version_packages value validator rejects a non-positive-integer package id at
// plan time. The error detail wraps at ~80 cols; the regex avoids whitespace at
// the wrap point by anchoring on the no-space token "positive".
func TestAccResource_ProPatchSoftwareTitle_InvalidPackageID(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "jamfplatform_pro_patch_software_title" "test" {
						name      = "tf-acc 8x8 Work invalid %[1]s"
						name_id   = %q
						source_id = %d

						version_packages = {
							%q = "0"
						}
					}
				`, suffix, accTitleNameID, accTitleSourceID, accTitleVersion),
				ExpectError: regexp.MustCompile(`positive`),
			},
		},
	})
}

// TestAccDataSource_ProPatchSoftwareTitle_ByID looks up a freshly-created title
// by ID.
func TestAccDataSource_ProPatchSoftwareTitle_ByID(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPatchSoftwareTitleDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "jamfplatform_pro_patch_software_title" "src" {
						name      = "tf-acc 8x8 Work ds %[1]s"
						name_id   = %q
						source_id = %d
					}

					data "jamfplatform_pro_patch_software_title" "lookup" {
						id = jamfplatform_pro_patch_software_title.src.id
					}
				`, suffix, accTitleNameID, accTitleSourceID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.jamfplatform_pro_patch_software_title.lookup", "name_id", "jamfplatform_pro_patch_software_title.src", "name_id"),
					resource.TestCheckResourceAttr("data.jamfplatform_pro_patch_software_title.lookup", "name_id", accTitleNameID),
					resource.TestCheckResourceAttrSet("data.jamfplatform_pro_patch_software_title.lookup", "available_versions.#"),
				),
			},
		},
	})
}

// TestAccListResource_ProPatchSoftwareTitle_Basic exercises the list resource via
// the `terraform query` workflow. The classic list endpoint surfaces no display
// name through the SDK, so DisplayName / the filter match on name_id.
func TestAccListResource_ProPatchSoftwareTitle_Basic(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()

	resource.Test(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_14_0),
		},
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPatchSoftwareTitleDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "jamfplatform_pro_patch_software_title" "src" {
						name      = "tf-acc 8x8 Work list %[1]s"
						name_id   = %q
						source_id = %d
					}
				`, suffix, accTitleNameID, accTitleSourceID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("jamfplatform_pro_patch_software_title.src", "id"),
				),
			},
			{
				Query: true,
				Config: fmt.Sprintf(`
					provider "jamfplatform" {}

					list "jamfplatform_pro_patch_software_title" "test" {
						provider         = jamfplatform
						include_resource = true

						config {
							filter = {
								name_substring = %q
							}
						}
					}
				`, accTitleNameID),
				QueryResultChecks: []querycheck.QueryResultCheck{
					querycheck.ExpectResourceKnownValues(
						"jamfplatform_pro_patch_software_title.test",
						queryfilter.ByDisplayName(knownvalue.StringExact(accTitleNameID)),
						[]querycheck.KnownValueCheck{
							{Path: tfjsonpath.New("name_id"), KnownValue: knownvalue.StringExact(accTitleNameID)},
						},
					),
				},
			},
		},
	})
}
