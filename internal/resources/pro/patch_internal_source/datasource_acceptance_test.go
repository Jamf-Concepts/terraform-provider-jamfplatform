// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

// Tests in this file read the Jamf ProClassic /patchinternalsources and
// /patchavailabletitles endpoints. Internal sources are Jamf-managed; every
// tenant ships the built-in "Jamf" source, which these tests look up. No
// resource is created or destroyed.

package patch_internal_source_test

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/testhelpers"
)

// TestAccDataSource_ProPatchInternalSource_ByName looks up the built-in "Jamf"
// internal source by name and asserts its metadata plus a populated
// available-titles catalog (the Jamf source publishes the full definitions set).
func TestAccDataSource_ProPatchInternalSource_ByName(t *testing.T) {
	testhelpers.AccPreCheck(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
					data "jamfplatform_pro_patch_internal_source" "jamf" {
						name = "Jamf"
					}
				`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.jamfplatform_pro_patch_internal_source.jamf", "id"),
					resource.TestCheckResourceAttr("data.jamfplatform_pro_patch_internal_source.jamf", "name", "Jamf"),
					resource.TestCheckResourceAttrSet("data.jamfplatform_pro_patch_internal_source.jamf", "enabled"),
					resource.TestCheckResourceAttrSet("data.jamfplatform_pro_patch_internal_source.jamf", "endpoint"),
					// The Jamf source publishes a large catalog: assert the field
					// is present and the first entry's name_id is set.
					resource.TestCheckResourceAttrSet("data.jamfplatform_pro_patch_internal_source.jamf", "available_titles.#"),
					resource.TestCheckResourceAttrSet("data.jamfplatform_pro_patch_internal_source.jamf", "available_titles.0.name_id"),
					resource.TestCheckResourceAttrSet("data.jamfplatform_pro_patch_internal_source.jamf", "available_titles.0.app_name"),
				),
			},
		},
	})
}

// TestAccDataSource_ProPatchInternalSource_ByID resolves the source by name in
// one data source, then looks it up again by the resolved id — exercising the
// by-id selector path.
func TestAccDataSource_ProPatchInternalSource_ByID(t *testing.T) {
	testhelpers.AccPreCheck(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
					data "jamfplatform_pro_patch_internal_source" "by_name" {
						name = "Jamf"
					}

					data "jamfplatform_pro_patch_internal_source" "by_id" {
						id = data.jamfplatform_pro_patch_internal_source.by_name.id
					}
				`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"data.jamfplatform_pro_patch_internal_source.by_id", "name",
						"data.jamfplatform_pro_patch_internal_source.by_name", "name",
					),
					resource.TestCheckResourceAttr("data.jamfplatform_pro_patch_internal_source.by_id", "name", "Jamf"),
					resource.TestCheckResourceAttrSet("data.jamfplatform_pro_patch_internal_source.by_id", "available_titles.#"),
				),
			},
		},
	})
}

// TestAccDataSource_ProPatchInternalSource_RequiresOneSelector asserts the
// ExactlyOneOf(id, name) config validator rejects a config supplying neither.
func TestAccDataSource_ProPatchInternalSource_RequiresOneSelector(t *testing.T) {
	testhelpers.AccPreCheck(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
					data "jamfplatform_pro_patch_internal_source" "none" {}
				`,
				ExpectError: regexp.MustCompile("Missing Attribute Configuration"),
			},
		},
	})
}
