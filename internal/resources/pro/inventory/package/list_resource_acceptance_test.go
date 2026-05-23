// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

package pkg_test

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

// TestAccListResource_ProPackage exercises the V1 /packages list endpoint
// via the `terraform query` workflow. The server-side filter restricts to
// `packageName=="<our run suffix>"` so the assertion targets one record
// even when other packages exist on the tenant.
func TestAccListResource_ProPackage(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-package-list-" + suffix
	fileName := name + ".pkg"

	resource.Test(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_14_0),
		},
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPackageDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "jamfplatform_pro_package" "src" {
  display_name = %q
  file_name    = %q
  info         = "list-acc-test"
}
`, name, fileName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("jamfplatform_pro_package.src", "id"),
				),
			},
			{
				Query: true,
				Config: fmt.Sprintf(`
provider "jamfplatform" {}

list "jamfplatform_pro_package" "test" {
  provider         = jamfplatform
  include_resource = true

  config {
    filter = [
      {
        selector  = "packageName"
        operator  = "=="
        argument  = %q
      },
    ]
  }
}
`, name),
				QueryResultChecks: []querycheck.QueryResultCheck{
					querycheck.ExpectLength("jamfplatform_pro_package.test", 1),
					querycheck.ExpectResourceKnownValues(
						"jamfplatform_pro_package.test",
						queryfilter.ByDisplayName(knownvalue.StringExact(name)),
						[]querycheck.KnownValueCheck{
							{Path: tfjsonpath.New("display_name"), KnownValue: knownvalue.StringExact(name)},
							{Path: tfjsonpath.New("file_name"), KnownValue: knownvalue.StringExact(fileName)},
						},
					),
				},
			},
		},
	})
}
