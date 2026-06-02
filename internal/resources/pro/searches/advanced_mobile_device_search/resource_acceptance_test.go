// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

// Tests in this file talk to the Jamf Pro /v1/advanced-mobile-device-searches
// endpoint.
//
// Design notes the acc run verifies (each is load-bearing — see the build spike):
//   - full-replace PUT: criteria/display_fields are removed by OMITTING the block,
//     never `= []`. Flatten returns null for an empty server array; a known empty
//     list ([]) would mismatch null and surface "inconsistent result after apply".
//     The clear step (step 5) drops both blocks and relies on the framework's
//     automatic post-apply empty-plan check to prove no perma-diff.
//   - GET-after-write: every Create/Update step implicitly verifies the GET-after
//     path (the Pro PUT response body echoes the submitted display fields and
//     silently drops invalid ones, so state must come from a fresh GET).
//   - display_fields is a Set: the server reorders columns, so positional checks
//     are avoided and TestCheckTypeSetElemAttr is used.

package advanced_mobile_device_search_test

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

const amdsResource = "jamfplatform_pro_advanced_mobile_device_search.test"

func testAccCheckAMDSDestroy(t *testing.T) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		c := pro.New(testhelpers.NewAcceptanceClient(t))
		ctx := context.Background()
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "jamfplatform_pro_advanced_mobile_device_search" {
				continue
			}
			_, err := c.GetAdvancedMobileDeviceSearchV1(ctx, rs.Primary.ID)
			if err != nil {
				if helpers.IsNotFoundError(err) {
					continue
				}
				return fmt.Errorf("error checking advanced mobile device search %s: %s", rs.Primary.ID, err)
			}
			return fmt.Errorf("advanced mobile device search %s still exists", rs.Primary.ID)
		}
		return nil
	}
}

// Step 1: create with 1 criterion + 2 display columns + site NONE.
func amdsConfigCreate(name string) string {
	return fmt.Sprintf(`
		resource "jamfplatform_pro_advanced_mobile_device_search" "test" {
			name = %q

			criteria = [
				{
					name        = "Managed"
					search_type = "is"
					value       = "Unmanaged"
				},
			]

			display_fields = ["Display Name", "Serial Number"]
		}
	`, name)
}

// Step 2: rename + grow criteria (1->2) + grow display (2->3).
func amdsConfigGrow(name string) string {
	return fmt.Sprintf(`
		resource "jamfplatform_pro_advanced_mobile_device_search" "test" {
			name = %q

			criteria = [
				{
					name        = "Managed"
					search_type = "is"
					value       = "Unmanaged"
				},
				{
					name        = "Supervised"
					search_type = "is"
					value       = "Unsupervised"
					and_or      = "and"
				},
			]

			display_fields = ["Display Name", "Serial Number", "Managed"]
		}
	`, name)
}

// Step 3: shrink criteria (2->1) and display (3->1) — within-collection replace.
func amdsConfigShrink(name string) string {
	return fmt.Sprintf(`
		resource "jamfplatform_pro_advanced_mobile_device_search" "test" {
			name = %q

			criteria = [
				{
					name        = "Managed"
					search_type = "is"
					value       = "Unmanaged"
				},
			]

			display_fields = ["Display Name"]
		}
	`, name)
}

// Step 4: clear by OMITTING criteria + display_fields (not `= []`). The implicit
// post-apply empty-plan check proves the full-replace clear round-trips with no
// perma-diff.
func amdsConfigCleared(name string) string {
	return fmt.Sprintf(`
		resource "jamfplatform_pro_advanced_mobile_device_search" "test" {
			name = %q
		}
	`, name)
}

func TestAccResource_ProAdvancedMobileDeviceSearch_Lifecycle(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-pro-amds-" + suffix
	renamed := "tf-acc-pro-amds-renamed-" + suffix

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckAMDSDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: amdsConfigCreate(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(amdsResource, "id"),
					resource.TestCheckResourceAttr(amdsResource, "name", name),
					resource.TestCheckResourceAttr(amdsResource, "site_id", "-1"),
					resource.TestCheckResourceAttr(amdsResource, "criteria.#", "1"),
					resource.TestCheckResourceAttr(amdsResource, "criteria.0.name", "Managed"),
					resource.TestCheckResourceAttr(amdsResource, "criteria.0.search_type", "is"),
					resource.TestCheckResourceAttr(amdsResource, "display_fields.#", "2"),
					resource.TestCheckTypeSetElemAttr(amdsResource, "display_fields.*", "Display Name"),
					resource.TestCheckTypeSetElemAttr(amdsResource, "display_fields.*", "Serial Number"),
				),
			},
			{
				Config: amdsConfigGrow(renamed),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(amdsResource, "name", renamed),
					resource.TestCheckResourceAttr(amdsResource, "criteria.#", "2"),
					resource.TestCheckResourceAttr(amdsResource, "criteria.1.name", "Supervised"),
					resource.TestCheckResourceAttr(amdsResource, "display_fields.#", "3"),
					resource.TestCheckTypeSetElemAttr(amdsResource, "display_fields.*", "Managed"),
				),
			},
			{
				// Import the POPULATED resource — the highest-risk round-trip.
				// Verifies server-assigned criterion priority and the reordered
				// display_fields Set survive import (Set compares order-independently).
				ResourceName:            amdsResource,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"timeouts"},
			},
			{
				Config: amdsConfigShrink(renamed),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(amdsResource, "criteria.#", "1"),
					resource.TestCheckResourceAttr(amdsResource, "display_fields.#", "1"),
				),
			},
			{
				Config: amdsConfigCleared(renamed),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(amdsResource, "name", renamed),
					// Cleared: omitted blocks flatten to null. The framework's
					// automatic post-apply empty-plan check is the real assertion.
					resource.TestCheckNoResourceAttr(amdsResource, "criteria.0.name"),
				),
			},
			{
				ResourceName:            amdsResource,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"timeouts"},
			},
		},
	})
}

// TestAccResource_ProAdvancedMobileDeviceSearch_EmptyNameRejected exercises the
// name LengthAtLeast(1) validator.
func TestAccResource_ProAdvancedMobileDeviceSearch_EmptyNameRejected(t *testing.T) {
	testhelpers.AccPreCheck(t)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
					resource "jamfplatform_pro_advanced_mobile_device_search" "test" {
						name = ""
					}
				`,
				ExpectError: regexp.MustCompile(`string length must be at least 1`),
			},
		},
	})
}

// TestAccResource_ProAdvancedMobileDeviceSearch_InvalidOperatorRejected exercises
// the search_type OneOf validator (full canonical vocabulary).
func TestAccResource_ProAdvancedMobileDeviceSearch_InvalidOperatorRejected(t *testing.T) {
	testhelpers.AccPreCheck(t)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
					resource "jamfplatform_pro_advanced_mobile_device_search" "test" {
						name = "tf-acc-bad-op"
						criteria = [
							{
								name        = "Managed"
								search_type = "not-a-real-operator"
								value       = "x"
							},
						]
					}
				`,
				ExpectError: regexp.MustCompile(`value must be one of`),
			},
		},
	})
}

func TestAccDataSource_ProAdvancedMobileDeviceSearch_ByIDAndName(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-pro-amds-ds-" + suffix

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckAMDSDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "jamfplatform_pro_advanced_mobile_device_search" "test" {
						name = %q

						criteria = [
							{
								name        = "Managed"
								search_type = "is"
								value       = "Unmanaged"
							},
						]

						display_fields = ["Display Name"]
					}

					data "jamfplatform_pro_advanced_mobile_device_search" "by_id" {
						id = jamfplatform_pro_advanced_mobile_device_search.test.id
					}

					data "jamfplatform_pro_advanced_mobile_device_search" "by_name" {
						name = jamfplatform_pro_advanced_mobile_device_search.test.name
					}
				`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.jamfplatform_pro_advanced_mobile_device_search.by_id", "name", amdsResource, "name"),
					resource.TestCheckResourceAttr("data.jamfplatform_pro_advanced_mobile_device_search.by_id", "criteria.#", "1"),
					resource.TestCheckResourceAttrPair("data.jamfplatform_pro_advanced_mobile_device_search.by_name", "id", amdsResource, "id"),
					resource.TestCheckResourceAttr("data.jamfplatform_pro_advanced_mobile_device_search.by_name", "display_fields.#", "1"),
				),
			},
		},
	})
}

func TestAccListResource_ProAdvancedMobileDeviceSearch_Basic(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-pro-amds-list-" + suffix

	resource.Test(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_14_0),
		},
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckAMDSDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "jamfplatform_pro_advanced_mobile_device_search" "test" {
						name = %q

						criteria = [
							{
								name        = "Managed"
								search_type = "is"
								value       = "Unmanaged"
							},
						]
					}
				`, name),
				Check: resource.TestCheckResourceAttrSet(amdsResource, "id"),
			},
			{
				Query: true,
				Config: fmt.Sprintf(`
					provider "jamfplatform" {}

					list "jamfplatform_pro_advanced_mobile_device_search" "test" {
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
					querycheck.ExpectLength("jamfplatform_pro_advanced_mobile_device_search.test", 1),
					querycheck.ExpectResourceKnownValues(
						"jamfplatform_pro_advanced_mobile_device_search.test",
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
