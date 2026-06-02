// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

// Tests in this file talk to the Jamf Pro /v1/advanced-user-content-searches
// endpoint (the admin UI calls these Advanced Volume Purchasing Content Searches).
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
//   - criteria/display-field names use Jamf Pro's WIRE vocabulary, which differs
//     from the UI labels (UI "Content Name" -> wire "Name", "Price" -> "Cost",
//     "Total Content" -> "Total", "Username" -> "Username").

package advanced_volume_purchasing_content_search_test

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

const avpcsResource = "jamfplatform_pro_advanced_volume_purchasing_content_search.test"

func testAccCheckAVPCSDestroy(t *testing.T) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		c := pro.New(testhelpers.NewAcceptanceClient(t))
		ctx := context.Background()
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "jamfplatform_pro_advanced_volume_purchasing_content_search" {
				continue
			}
			_, err := c.GetAdvancedUserContentSearchV1(ctx, rs.Primary.ID)
			if err != nil {
				if helpers.IsNotFoundError(err) {
					continue
				}
				return fmt.Errorf("error checking advanced volume purchasing content search %s: %s", rs.Primary.ID, err)
			}
			return fmt.Errorf("advanced volume purchasing content search %s still exists", rs.Primary.ID)
		}
		return nil
	}
}

// Step 1: create with 1 criterion + 2 display columns + site NONE.
func avpcsConfigCreate(name string) string {
	return fmt.Sprintf(`
		resource "jamfplatform_pro_advanced_volume_purchasing_content_search" "test" {
			name = %q

			criteria = [
				{
					name        = "Name"
					search_type = "like"
					value       = "Office"
				},
			]

			display_fields = ["Name", "Cost"]
		}
	`, name)
}

// Step 2: rename + grow criteria (1->2) + grow display (2->3).
func avpcsConfigGrow(name string) string {
	return fmt.Sprintf(`
		resource "jamfplatform_pro_advanced_volume_purchasing_content_search" "test" {
			name = %q

			criteria = [
				{
					name        = "Name"
					search_type = "like"
					value       = "Office"
				},
				{
					name        = "Username"
					search_type = "is"
					value       = "alice"
					and_or      = "and"
				},
			]

			display_fields = ["Name", "Cost", "Total"]
		}
	`, name)
}

// Step 3: shrink criteria (2->1) and display (3->1) — within-collection replace.
func avpcsConfigShrink(name string) string {
	return fmt.Sprintf(`
		resource "jamfplatform_pro_advanced_volume_purchasing_content_search" "test" {
			name = %q

			criteria = [
				{
					name        = "Name"
					search_type = "like"
					value       = "Office"
				},
			]

			display_fields = ["Name"]
		}
	`, name)
}

// Step 4: clear by OMITTING criteria + display_fields (not `= []`). The implicit
// post-apply empty-plan check proves the full-replace clear round-trips with no
// perma-diff.
func avpcsConfigCleared(name string) string {
	return fmt.Sprintf(`
		resource "jamfplatform_pro_advanced_volume_purchasing_content_search" "test" {
			name = %q
		}
	`, name)
}

func TestAccResource_ProAdvancedVolumePurchasingContentSearch_Lifecycle(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-pro-avpcs-" + suffix
	renamed := "tf-acc-pro-avpcs-renamed-" + suffix

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckAVPCSDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: avpcsConfigCreate(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(avpcsResource, "id"),
					resource.TestCheckResourceAttr(avpcsResource, "name", name),
					resource.TestCheckResourceAttr(avpcsResource, "site_id", "-1"),
					resource.TestCheckResourceAttr(avpcsResource, "criteria.#", "1"),
					resource.TestCheckResourceAttr(avpcsResource, "criteria.0.name", "Name"),
					resource.TestCheckResourceAttr(avpcsResource, "criteria.0.search_type", "like"),
					resource.TestCheckResourceAttr(avpcsResource, "display_fields.#", "2"),
					resource.TestCheckTypeSetElemAttr(avpcsResource, "display_fields.*", "Name"),
					resource.TestCheckTypeSetElemAttr(avpcsResource, "display_fields.*", "Cost"),
				),
			},
			{
				Config: avpcsConfigGrow(renamed),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(avpcsResource, "name", renamed),
					resource.TestCheckResourceAttr(avpcsResource, "criteria.#", "2"),
					resource.TestCheckResourceAttr(avpcsResource, "criteria.1.name", "Username"),
					resource.TestCheckResourceAttr(avpcsResource, "display_fields.#", "3"),
					resource.TestCheckTypeSetElemAttr(avpcsResource, "display_fields.*", "Total"),
				),
			},
			{
				// Import the POPULATED resource — the highest-risk round-trip.
				// Verifies server-assigned criterion priority and the reordered
				// display_fields Set survive import (Set compares order-independently).
				ResourceName:            avpcsResource,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"timeouts"},
			},
			{
				Config: avpcsConfigShrink(renamed),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(avpcsResource, "criteria.#", "1"),
					resource.TestCheckResourceAttr(avpcsResource, "display_fields.#", "1"),
				),
			},
			{
				Config: avpcsConfigCleared(renamed),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(avpcsResource, "name", renamed),
					// Cleared: omitted blocks flatten to null. The framework's
					// automatic post-apply empty-plan check is the real assertion.
					resource.TestCheckNoResourceAttr(avpcsResource, "criteria.0.name"),
				),
			},
			{
				ResourceName:            avpcsResource,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"timeouts"},
			},
		},
	})
}

// TestAccResource_ProAdvancedVolumePurchasingContentSearch_EmptyNameRejected
// exercises the name LengthAtLeast(1) validator.
func TestAccResource_ProAdvancedVolumePurchasingContentSearch_EmptyNameRejected(t *testing.T) {
	testhelpers.AccPreCheck(t)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
					resource "jamfplatform_pro_advanced_volume_purchasing_content_search" "test" {
						name = ""
					}
				`,
				ExpectError: regexp.MustCompile(`string length must be at least 1`),
			},
		},
	})
}

// TestAccResource_ProAdvancedVolumePurchasingContentSearch_SubsetOperatorRejected
// exercises the content operator subset: `member of` is a valid canonical operator
// but is excluded from the Volume-Purchasing-Content vocabulary, so the provider
// must reject it at plan time (proving the criteria.Without subset is wired up).
func TestAccResource_ProAdvancedVolumePurchasingContentSearch_SubsetOperatorRejected(t *testing.T) {
	testhelpers.AccPreCheck(t)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
					resource "jamfplatform_pro_advanced_volume_purchasing_content_search" "test" {
						name = "tf-acc-bad-op"
						criteria = [
							{
								name        = "Name"
								search_type = "member of"
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

func TestAccDataSource_ProAdvancedVolumePurchasingContentSearch_ByIDAndName(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-pro-avpcs-ds-" + suffix

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckAVPCSDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "jamfplatform_pro_advanced_volume_purchasing_content_search" "test" {
						name = %q

						criteria = [
							{
								name        = "Name"
								search_type = "like"
								value       = "Office"
							},
						]

						display_fields = ["Name"]
					}

					data "jamfplatform_pro_advanced_volume_purchasing_content_search" "by_id" {
						id = jamfplatform_pro_advanced_volume_purchasing_content_search.test.id
					}

					data "jamfplatform_pro_advanced_volume_purchasing_content_search" "by_name" {
						name = jamfplatform_pro_advanced_volume_purchasing_content_search.test.name
					}
				`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.jamfplatform_pro_advanced_volume_purchasing_content_search.by_id", "name", avpcsResource, "name"),
					resource.TestCheckResourceAttr("data.jamfplatform_pro_advanced_volume_purchasing_content_search.by_id", "criteria.#", "1"),
					resource.TestCheckResourceAttrPair("data.jamfplatform_pro_advanced_volume_purchasing_content_search.by_name", "id", avpcsResource, "id"),
					resource.TestCheckResourceAttr("data.jamfplatform_pro_advanced_volume_purchasing_content_search.by_name", "display_fields.#", "1"),
				),
			},
		},
	})
}

func TestAccListResource_ProAdvancedVolumePurchasingContentSearch_Basic(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-pro-avpcs-list-" + suffix

	resource.Test(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_14_0),
		},
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckAVPCSDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "jamfplatform_pro_advanced_volume_purchasing_content_search" "test" {
						name = %q

						criteria = [
							{
								name        = "Name"
								search_type = "like"
								value       = "Office"
							},
						]
					}
				`, name),
				Check: resource.TestCheckResourceAttrSet(avpcsResource, "id"),
			},
			{
				Query: true,
				Config: fmt.Sprintf(`
					provider "jamfplatform" {}

					list "jamfplatform_pro_advanced_volume_purchasing_content_search" "test" {
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
					querycheck.ExpectLength("jamfplatform_pro_advanced_volume_purchasing_content_search.test", 1),
					querycheck.ExpectResourceKnownValues(
						"jamfplatform_pro_advanced_volume_purchasing_content_search.test",
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
