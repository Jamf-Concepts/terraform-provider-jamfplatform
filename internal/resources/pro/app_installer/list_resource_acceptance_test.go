// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

// Tests in this file exercise the jamfplatform_pro_app_installer list resource
// through the `terraform query` workflow, and specifically the include_resource
// path that Terraform's configuration generation drives.
//
// That path is worth its own file because it is the one issue #379 broke the
// hardest. A deployment reports the App Catalog title it deploys only as an id,
// while app_title_name is schema-Required, so a listing that hydrates from the
// deployment read alone produces `app_title_name = null` — and Terraform answers a
// null Required attribute by failing the WHOLE run, taking every other resource
// type in the query down with it. A tenant holding one App Installer could not
// generate configuration for anything.
package app_installer_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/querycheck"
	"github.com/hashicorp/terraform-plugin-testing/querycheck/queryfilter"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/testhelpers"
)

// TestAccListResource_ProAppInstaller_IncludeResourceNamesTheTitle creates a
// deployment, then queries the list resource with include_resource so the
// hydrated resource object is returned, and asserts the App Catalog title is
// named rather than left as a bare id.
//
// The catalog name is asserted against the catalog data source's own resolved
// value in step 1 rather than restated here, so the two directions of the
// mapping have to agree.
//
// app_title_name in step 2 is the assertion the whole file exists for: a null
// there is what failed the entire query before issue #379, because the attribute
// is Required.
func TestAccListResource_ProAppInstaller_IncludeResourceNamesTheTitle(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-appinst-list-" + suffix

	resource.Test(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_14_0),
		},
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckAppInstallerDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(catalogDS+`
					resource "jamfplatform_pro_app_installer" "test" {
						name            = %q
						app_title_name  = %s
						deployment_type = "SELF_SERVICE"
						update_behavior = "AUTOMATIC"

						category_id    = "-1"
						site_id        = "-1"
						smart_group_id = "-1"
					}
				`, name, titleNameRef),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceAddr, "app_title_name", titleName),
					resource.TestCheckResourceAttrSet(resourceAddr, "app_title_id"),
				),
			},
			{
				Query: true,
				Config: fmt.Sprintf(`
					provider "jamfplatform" {}

					list "jamfplatform_pro_app_installer" "test" {
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
					querycheck.ExpectLength("jamfplatform_pro_app_installer.test", 1),
					querycheck.ExpectResourceKnownValues(
						"jamfplatform_pro_app_installer.test",
						queryfilter.ByDisplayName(knownvalue.StringExact(name)),
						[]querycheck.KnownValueCheck{
							{Path: tfjsonpath.New("name"), KnownValue: knownvalue.StringExact(name)},
							{Path: tfjsonpath.New("app_title_name"), KnownValue: knownvalue.StringExact(titleName)},
						},
					),
				},
			},
		},
	})
}
