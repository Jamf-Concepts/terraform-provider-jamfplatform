// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

// Tests in this file talk to the Jamf Pro v3 patch-software-title-configurations
// endpoints, plus the one classic /patchsoftwaretitles call that mints a title's
// id (see crud.go). That classic create has known concurrency issues when
// multiple writes hit the same resource type — keep these tests serial with any
// future classic acceptance work in this package.
//
// The fixtures use the real "8x8 Work" patch catalog entry: name_id "285",
// source_id 1, with versions "8.33.2.2" and "8.32.2.10". If the catalog entry or
// either version is removed from the test tenant's patch source, these tests
// will need updating.

package patch_software_title_test

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/pro"
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
	accTitleNameID     = "285" // 8x8 Work
	accTitleSourceID   = 1
	accTitleVersion    = "8.33.2.2"
	accTitleVersionAlt = "8.32.2.10"
	patchSoftwareType  = "jamfplatform_pro_patch_software_title"
)

// testAccCheckPatchSoftwareTitleDestroy verifies titles created during the test
// were destroyed.
//
// The read goes through the same v3 configurations endpoint the resource itself
// deletes through. Wire-probed 2026-09-02: the v3 DELETE removes the classic
// title with it, so a 404 here means the object is gone from both surfaces.
func testAccCheckPatchSoftwareTitleDestroy(t *testing.T) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		c := pro.New(testhelpers.NewAcceptanceClient(t))
		ctx := context.Background()

		for _, rs := range s.RootModule().Resources {
			if rs.Type != patchSoftwareType {
				continue
			}
			_, err := c.GetPatchSoftwareTitleConfigurationV3(ctx, rs.Primary.ID)
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

	// Step 3: point the same version at a different package, exercising the
	// re-assign path through the v3 packages array.
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

			# Re-point version A at package B to prove the assign path mutates in
			# place rather than accumulating.
			version_packages = {
				%q = jamfplatform_pro_package.pkg_b.id
			}
		}
	`, cat, site, pkgA, pkgA, pkgB, pkgB, suffix, accTitleNameID, accTitleSourceID, accTitleVersion)

	// Step 4: remove the version_packages key entirely → exercises the unassign
	// path, where the key drops out of the replacement array Update sends.
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
					// Server defaults. v3 reports an unassigned category or site as
					// the literal "-1", the only non-positive id it accepts on a
					// write, so both are always known.
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
					// version_packages fully cleared (unassign path).
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
				// available_versions: server-derived, and at this step the state
				// already matches. source_id is resolved from the patch source
				// name on import, so it round-trips.
				ImportStateVerifyIgnore: []string{"timeouts", "version_packages"},
			},
		},
	})
}

// TestAccResource_ProPatchSoftwareTitle_OutOfBandAssignmentSurvives pins the
// managed-subset contract version_packages documents: only the versions
// Terraform declares are managed, and a package an admin assigns to some other
// version through the UI must still be there after the next apply.
//
// It earns an acceptance test because the v3 packages array is a full
// replacement — the naive migration, sending the plan's assignments alone,
// silently wipes every assignment Terraform does not know about, and no unit
// test over the request builder can catch that. Step two mutates the title
// outside Terraform, then forces an Update by renaming, so the read-modify-write
// fold in crud.go Update is what keeps the out-of-band assignment alive.
func TestAccResource_ProPatchSoftwareTitle_OutOfBandAssignmentSurvives(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()

	pkgA := "tf-acc-pst-oob-pkgA-" + suffix
	pkgB := "tf-acc-pst-oob-pkgB-" + suffix

	var titleID, pkgBID string

	config := func(name string) string {
		return fmt.Sprintf(`
		resource "jamfplatform_pro_package" "pkg_a" {
			display_name = %q
			file_name    = "%s.pkg"
		}

		resource "jamfplatform_pro_package" "pkg_b" {
			display_name = %q
			file_name    = "%s.pkg"
		}

		resource "jamfplatform_pro_patch_software_title" "test" {
			name      = %q
			name_id   = %q
			source_id = %d

			version_packages = {
				%q = jamfplatform_pro_package.pkg_a.id
			}
		}
	`, pkgA, pkgA, pkgB, pkgB, name, accTitleNameID, accTitleSourceID, accTitleVersion)
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPatchSoftwareTitleDestroy(t),
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_13_0),
		},
		Steps: []resource.TestStep{
			{
				Config: config("tf-acc 8x8 Work oob " + suffix),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(patchSoftwareType+".test", "version_packages."+accTitleVersion, "jamfplatform_pro_package.pkg_a", "id"),
					testAccCapturePatchTitleAndPackage(&titleID, &pkgBID),
				),
			},
			{
				PreConfig: func() {
					if err := testAccAssignPatchPackageOutOfBand(t, titleID, accTitleVersionAlt, pkgBID); err != nil {
						t.Fatalf("assigning a package outside Terraform: %v", err)
					}
				},
				Config: config("tf-acc 8x8 Work oob renamed " + suffix),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(patchSoftwareType+".test", "name", "tf-acc 8x8 Work oob renamed "+suffix),
					// The managed key is unchanged and still the only one in state.
					resource.TestCheckResourceAttrPair(patchSoftwareType+".test", "version_packages."+accTitleVersion, "jamfplatform_pro_package.pkg_a", "id"),
					resource.TestCheckResourceAttr(patchSoftwareType+".test", "version_packages.%", "1"),
					// The unmanaged key is untouched on the server.
					testAccCheckPatchTitleAssignment(t, &titleID, accTitleVersionAlt, &pkgBID),
				),
			},
		},
	})
}

// testAccCapturePatchTitleAndPackage records the title id and the id of the
// package the next step assigns outside Terraform. PreConfig takes no state, so
// the ids have to be captured from a Check in the preceding step.
func testAccCapturePatchTitleAndPackage(titleID, pkgBID *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		title, ok := s.RootModule().Resources[patchSoftwareType+".test"]
		if !ok {
			return fmt.Errorf("patch software title not found in state")
		}
		pkg, ok := s.RootModule().Resources["jamfplatform_pro_package.pkg_b"]
		if !ok {
			return fmt.Errorf("package pkg_b not found in state")
		}
		*titleID = title.Primary.ID
		*pkgBID = pkg.Primary.ID
		return nil
	}
}

// testAccAssignPatchPackageOutOfBand assigns a package to one version of a title
// the way an administrator would in the UI: a v3 merge-patch the provider had no
// part in. It reads the current assignments first and appends, because the
// packages array replaces rather than merges.
func testAccAssignPatchPackageOutOfBand(t *testing.T, titleID, version, packageID string) error {
	t.Helper()
	if titleID == "" || packageID == "" {
		return fmt.Errorf("nothing captured from the previous step (title %q, package %q)", titleID, packageID)
	}

	c := pro.New(testhelpers.NewAcceptanceClient(t))
	ctx := context.Background()

	current, err := c.GetPatchSoftwareTitleConfigurationV3(ctx, titleID)
	if err != nil {
		return fmt.Errorf("reading title %s: %w", titleID, err)
	}

	packages := append([]pro.PatchSoftwareTitlePackages{}, current.Packages...)
	v, p := version, packageID
	packages = append(packages, pro.PatchSoftwareTitlePackages{Version: &v, PackageID: &p})

	if _, err := c.UpdatePatchSoftwareTitleConfigurationV3(ctx, titleID, &pro.PatchSoftwareTitleConfigurationPatch{
		Packages: &packages,
	}); err != nil {
		return fmt.Errorf("assigning package %s to version %s of title %s: %w", packageID, version, titleID, err)
	}
	return nil
}

// testAccCheckPatchTitleAssignment asserts the server still reports the given
// version pointing at the given package. The ids are read through pointers
// because they are only known once an earlier step has run.
func testAccCheckPatchTitleAssignment(t *testing.T, titleID *string, version string, packageID *string) resource.TestCheckFunc {
	t.Helper()
	return func(*terraform.State) error {
		c := pro.New(testhelpers.NewAcceptanceClient(t))
		got, err := c.GetPatchSoftwareTitleConfigurationV3(context.Background(), *titleID)
		if err != nil {
			return fmt.Errorf("reading title %s: %w", *titleID, err)
		}
		for _, pkg := range got.Packages {
			if pkg.Version == nil || *pkg.Version != version {
				continue
			}
			if pkg.PackageID == nil || *pkg.PackageID != *packageID {
				return fmt.Errorf("version %s points at package %v, want %s", version, pkg.PackageID, *packageID)
			}
			return nil
		}
		return fmt.Errorf("version %s has no package assignment; the apply wiped an assignment Terraform did not manage", version)
	}
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
func TestAccDataSource_ProPatchSoftwareTitle_ByIDAndName(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := fmt.Sprintf("tf-acc 8x8 Work ds %s", suffix)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPatchSoftwareTitleDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "jamfplatform_pro_patch_software_title" "src" {
						name      = %q
						name_id   = %q
						source_id = %d
					}

					data "jamfplatform_pro_patch_software_title" "by_id" {
						id = jamfplatform_pro_patch_software_title.src.id
					}

					data "jamfplatform_pro_patch_software_title" "by_name" {
						name = jamfplatform_pro_patch_software_title.src.name
					}
				`, name, accTitleNameID, accTitleSourceID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.jamfplatform_pro_patch_software_title.by_id", "name_id", "jamfplatform_pro_patch_software_title.src", "name_id"),
					resource.TestCheckResourceAttr("data.jamfplatform_pro_patch_software_title.by_id", "name_id", accTitleNameID),
					resource.TestCheckResourceAttr("data.jamfplatform_pro_patch_software_title.by_id", "name", name),
					resource.TestCheckResourceAttrSet("data.jamfplatform_pro_patch_software_title.by_id", "available_versions.#"),

					resource.TestCheckResourceAttrPair("data.jamfplatform_pro_patch_software_title.by_name", "id", "jamfplatform_pro_patch_software_title.src", "id"),
					resource.TestCheckResourceAttr("data.jamfplatform_pro_patch_software_title.by_name", "name_id", accTitleNameID),
					resource.TestCheckResourceAttrSet("data.jamfplatform_pro_patch_software_title.by_name", "available_versions.#"),
				),
			},
		},
	})
}

// TestAccListResource_ProPatchSoftwareTitle_Basic exercises the list resource via
// the `terraform query` workflow. DisplayName / the filter match on the title's
// display name.
func TestAccListResource_ProPatchSoftwareTitle_Basic(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := fmt.Sprintf("tf-acc 8x8 Work list %s", suffix)

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
						name      = %q
						name_id   = %q
						source_id = %d
					}
				`, name, accTitleNameID, accTitleSourceID),
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
				`, name),
				QueryResultChecks: []querycheck.QueryResultCheck{
					querycheck.ExpectResourceKnownValues(
						"jamfplatform_pro_patch_software_title.test",
						queryfilter.ByDisplayName(knownvalue.StringExact(name)),
						[]querycheck.KnownValueCheck{
							{Path: tfjsonpath.New("name"), KnownValue: knownvalue.StringExact(name)},
							{Path: tfjsonpath.New("name_id"), KnownValue: knownvalue.StringExact(accTitleNameID)},
						},
					),
				},
			},
		},
	})
}

// TestAccResource_ProPatchSoftwareTitle_ExtensionAttributeAccept exercises the
// v2 extension-attribute side-channel. "Adobe AIR" (name_id 0AE) is a title for
// which Jamf supplies an extension attribute that must be accepted. Step 1
// creates the title without accepting (extension_attributes present, accepted =
// false); step 2 sets accept_extension_attributes = true and asserts the EA
// flips to accepted = true. Accepting is one-way, so there is no revert step.
func TestAccResource_ProPatchSoftwareTitle_ExtensionAttributeAccept(t *testing.T) {
	testhelpers.AccPreCheck(t)

	const (
		eaTitleNameID   = "0AE" // Adobe AIR
		eaTitleSourceID = 1
	)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPatchSoftwareTitleDestroy(t),
		Steps: []resource.TestStep{
			{
				// Create without accepting — the EA is present but not accepted.
				Config: fmt.Sprintf(`
					resource "jamfplatform_pro_patch_software_title" "ea" {
						name      = "Adobe AIR"
						name_id   = %q
						source_id = %d
					}
				`, eaTitleNameID, eaTitleSourceID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(patchSoftwareType+".ea", "extension_attributes.#", "1"),
					resource.TestCheckResourceAttr(patchSoftwareType+".ea", "extension_attributes.0.accepted", "false"),
					resource.TestCheckResourceAttrSet(patchSoftwareType+".ea", "extension_attributes.0.ea_id"),
					resource.TestCheckResourceAttrSet(patchSoftwareType+".ea", "extension_attributes.0.display_name"),
				),
			},
			{
				// Accept — the EA flips to accepted = true.
				Config: fmt.Sprintf(`
					resource "jamfplatform_pro_patch_software_title" "ea" {
						name                        = "Adobe AIR"
						name_id                     = %q
						source_id                   = %d
						accept_extension_attributes = true
					}
				`, eaTitleNameID, eaTitleSourceID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(patchSoftwareType+".ea", "accept_extension_attributes", "true"),
					resource.TestCheckResourceAttr(patchSoftwareType+".ea", "extension_attributes.#", "1"),
					resource.TestCheckResourceAttr(patchSoftwareType+".ea", "extension_attributes.0.accepted", "true"),
				),
			},
		},
	})
}
