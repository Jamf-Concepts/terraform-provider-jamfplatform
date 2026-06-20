// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

package app_installer_titles_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/testhelpers"
)

// TestAccDataSource_ProAppInstallerTitles_All reads the whole catalog and
// asserts at least one title is returned.
func TestAccDataSource_ProAppInstallerTitles_All(t *testing.T) {
	testhelpers.AccPreCheck(t)
	cfg := `data "jamfplatform_pro_app_installer_titles" "all" {}`

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: cfg,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttrSet("data.jamfplatform_pro_app_installer_titles.all", "titles.#"),
				resource.TestCheckResourceAttrSet("data.jamfplatform_pro_app_installer_titles.all", "titles.0.id"),
				resource.TestCheckResourceAttrSet("data.jamfplatform_pro_app_installer_titles.all", "titles.0.title_name"),
			),
		}},
	})
}

// TestAccDataSource_ProAppInstallerTitles_Substring narrows the catalog by name.
func TestAccDataSource_ProAppInstallerTitles_Substring(t *testing.T) {
	testhelpers.AccPreCheck(t)
	cfg := `
		data "jamfplatform_pro_app_installer_titles" "filtered" {
			name_substring = "010 Editor"
		}
	`
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: cfg,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("data.jamfplatform_pro_app_installer_titles.filtered", "titles.#", "1"),
			),
		}},
	})
}
