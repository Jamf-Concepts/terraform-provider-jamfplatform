// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

package app_installer_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/testhelpers"
)

// TestAccDataSource_ProAppInstallers_All creates a deployment, then lists all
// deployments through the plural data source and asserts the expanded shape is
// populated (the created deployment's app ref resolves).
func TestAccDataSource_ProAppInstallers_All(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-app-installers-ds-" + suffix
	cfg := fmt.Sprintf(`
		data "jamfplatform_pro_app_installer_titles" "catalog" {
			name_substring = "010 Editor"
		}
		resource "jamfplatform_pro_app_installer" "test" {
			name            = %q
			app_title_name  = data.jamfplatform_pro_app_installer_titles.catalog.titles[0].title_name
			deployment_type = "SELF_SERVICE"
			update_behavior = "AUTOMATIC"
		}
		data "jamfplatform_pro_app_installers" "all" {
			name_substring = %q
			depends_on     = [jamfplatform_pro_app_installer.test]
		}
	`, name, name)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: cfg,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("data.jamfplatform_pro_app_installers.all", "deployments.#", "1"),
				resource.TestCheckResourceAttr("data.jamfplatform_pro_app_installers.all", "deployments.0.name", name),
				resource.TestCheckResourceAttrSet("data.jamfplatform_pro_app_installers.all", "deployments.0.app.id"),
				resource.TestCheckResourceAttrSet("data.jamfplatform_pro_app_installers.all", "deployments.0.update_behavior"),
			),
		}},
	})
}
