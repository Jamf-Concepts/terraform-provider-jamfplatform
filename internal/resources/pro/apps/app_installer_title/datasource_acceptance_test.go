// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

package app_installer_title_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/testhelpers"
)

// TestAccDataSource_ProAppInstallerTitle_ByID resolves a title ID from the
// plural catalog data source, then reads it by ID through the singular data
// source and asserts the core fields are populated.
func TestAccDataSource_ProAppInstallerTitle_ByID(t *testing.T) {
	testhelpers.AccPreCheck(t)
	cfg := `
		data "jamfplatform_pro_app_installer_titles" "catalog" {
			name_substring = "010 Editor"
		}
		data "jamfplatform_pro_app_installer_title" "one" {
			id = data.jamfplatform_pro_app_installer_titles.catalog.titles[0].id
		}
	`
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: cfg,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttrSet("data.jamfplatform_pro_app_installer_title.one", "id"),
				resource.TestCheckResourceAttrSet("data.jamfplatform_pro_app_installer_title.one", "title_name"),
				resource.TestCheckResourceAttrSet("data.jamfplatform_pro_app_installer_title.one", "bundle_id"),
			),
		}},
	})
}
