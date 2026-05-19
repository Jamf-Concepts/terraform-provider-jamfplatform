// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

package category_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/pro"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/testhelpers"
)

// testAccCheckCategoryDestroy verifies categories created during the test were destroyed.
func testAccCheckCategoryDestroy(t *testing.T) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		c := pro.New(testhelpers.NewAcceptanceClient(t))
		ctx := context.Background()

		for _, rs := range s.RootModule().Resources {
			if rs.Type != "jamfplatform_pro_category" {
				continue
			}
			_, err := c.GetCategoryV1(ctx, rs.Primary.ID)
			if err != nil {
				if helpers.IsNotFoundError(err) {
					continue
				}
				return fmt.Errorf("error checking Jamf Pro category %s: %s", rs.Primary.ID, err)
			}
			return fmt.Errorf("Jamf Pro category %s still exists", rs.Primary.ID)
		}
		return nil
	}
}

func TestAccResource_ProCategory_Basic(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-pro-category-" + suffix
	nameUpdated := "tf-acc-pro-category-updated-" + suffix

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckCategoryDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "jamfplatform_pro_category" "test" {
						name     = %q
						priority = 7
					}
				`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("jamfplatform_pro_category.test", "id"),
					resource.TestCheckResourceAttr("jamfplatform_pro_category.test", "name", name),
					resource.TestCheckResourceAttr("jamfplatform_pro_category.test", "priority", "7"),
				),
			},
			{
				Config: fmt.Sprintf(`
					resource "jamfplatform_pro_category" "test" {
						name     = %q
						priority = 3
					}
				`, nameUpdated),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("jamfplatform_pro_category.test", "name", nameUpdated),
					resource.TestCheckResourceAttr("jamfplatform_pro_category.test", "priority", "3"),
				),
			},
			{
				ResourceName:      "jamfplatform_pro_category.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"timeouts",
				},
			},
		},
	})
}

func TestAccDataSource_ProCategory_Basic(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-pro-category-ds-" + suffix

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckCategoryDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "jamfplatform_pro_category" "src" {
						name     = %q
						priority = 11
					}

					data "jamfplatform_pro_category" "lookup" {
						id = jamfplatform_pro_category.src.id
					}
				`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.jamfplatform_pro_category.lookup", "name", "jamfplatform_pro_category.src", "name"),
					resource.TestCheckResourceAttrPair("data.jamfplatform_pro_category.lookup", "priority", "jamfplatform_pro_category.src", "priority"),
				),
			},
		},
	})
}

func TestAccDataSource_ProCategories_FilterByName(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-pro-categories-" + suffix

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckCategoryDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "jamfplatform_pro_category" "src" {
						name     = %q
						priority = 9
					}

					data "jamfplatform_pro_categories" "lookup" {
						filter = [
							{
								selector = "name"
								argument = jamfplatform_pro_category.src.name
							}
						]
					}
				`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.jamfplatform_pro_categories.lookup", "categories.#", "1"),
					resource.TestCheckResourceAttr("data.jamfplatform_pro_categories.lookup", "categories.0.name", name),
				),
			},
		},
	})
}
