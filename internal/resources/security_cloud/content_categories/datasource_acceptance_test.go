// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

package content_categories_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/testhelpers"
)

// TestAccDataSource_SecurityCloudContentCategories_ReadsCatalogue reads the
// Jamf-curated catalogue and asserts it is non-empty and shaped as expected.
//
// It asserts presence rather than specific entries. The catalogue is Jamf's, not
// the tenant's — its contents change when Jamf revises the taxonomy, and pinning
// individual IDs or names would make the test a hostage to that. The fixture helper
// already skips when the tenant cannot read it at all, so an empty result here
// would mean the read succeeded and returned nothing, which is worth failing on.
func TestAccDataSource_SecurityCloudContentCategories_ReadsCatalogue(t *testing.T) {
	testhelpers.AccPreCheckSecurityCloud(t)
	testhelpers.RequireSecurityCloudContentCategories(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `data "jamfplatform_security_cloud_content_categories" "all" {}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.jamfplatform_security_cloud_content_categories.all", "id", "content_categories"),
					resource.TestCheckResourceAttrSet("data.jamfplatform_security_cloud_content_categories.all", "content_categories.#"),
					resource.TestCheckResourceAttrSet("data.jamfplatform_security_cloud_content_categories.all", "content_categories.0.id"),
					resource.TestCheckResourceAttrSet("data.jamfplatform_security_cloud_content_categories.all", "content_categories.0.display_name"),
					resource.TestCheckResourceAttrSet("data.jamfplatform_security_cloud_content_categories.all", "content_categories.0.name"),
				),
			},
		},
	})
}

// TestAccDataSource_SecurityCloudContentCategories_DisplayNameDiffersFromName pins
// the distinction this data source exists to make visible. The two names are not
// interchangeable — a Zero Trust Network Access app's category must match
// `display_name`, and passing `name` is refused server-side with a code that names
// neither field. If the API ever collapses them, the description telling readers to
// pick one becomes misleading, and this is where that shows up.
func TestAccDataSource_SecurityCloudContentCategories_DisplayNameDiffersFromName(t *testing.T) {
	testhelpers.AccPreCheckSecurityCloud(t)
	categories := testhelpers.RequireSecurityCloudContentCategories(t)

	first := categories[0]
	if first.Name == first.DisplayName {
		t.Skipf("Skipping: tenant's first category has identical name and displayName (%q); nothing to distinguish", first.Name)
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
					data "jamfplatform_security_cloud_content_categories" "all" {}

					locals {
						matched = one([
							for category in data.jamfplatform_security_cloud_content_categories.all.content_categories :
							category if category.id == "` + first.ID + `"
						])
					}

					output "matched_display_name" {
						value = local.matched.display_name
					}

					output "matched_name" {
						value = local.matched.name
					}
				`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckOutput("matched_display_name", first.DisplayName),
					resource.TestCheckOutput("matched_name", first.Name),
				),
			},
		},
	})
}
