// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

package pkg_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/testhelpers"
)

// TestAccDataSource_ProPackage_ByID looks up a package by its server-assigned
// ID via a fixture resource block in the same configuration.
func TestAccDataSource_ProPackage_ByID(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-package-ds-id-" + suffix
	fileName := name + ".pkg"

	dsAddr := "data.jamfplatform_pro_package.by_id"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPackageDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "jamfplatform_pro_package" "test" {
  display_name = %q
  file_name    = %q
  info         = "ds-by-id"
}

data "jamfplatform_pro_package" "by_id" {
  id = jamfplatform_pro_package.test.id
}
`, name, fileName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(dsAddr, "id", packageResourceAddr, "id"),
					resource.TestCheckResourceAttr(dsAddr, "display_name", name),
					resource.TestCheckResourceAttr(dsAddr, "file_name", fileName),
					resource.TestCheckResourceAttr(dsAddr, "info", "ds-by-id"),
				),
			},
		},
	})
}

// TestAccDataSource_ProPackage_ByName looks up a package by its display_name
// (wire field `packageName`) via ResolvePackageV1ByName.
func TestAccDataSource_ProPackage_ByName(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-package-ds-name-" + suffix
	fileName := name + ".pkg"

	dsAddr := "data.jamfplatform_pro_package.by_name"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPackageDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "jamfplatform_pro_package" "test" {
  display_name = %q
  file_name    = %q
  info         = "ds-by-name"
}

data "jamfplatform_pro_package" "by_name" {
  display_name = jamfplatform_pro_package.test.display_name
}
`, name, fileName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(dsAddr, "id", packageResourceAddr, "id"),
					resource.TestCheckResourceAttr(dsAddr, "display_name", name),
					resource.TestCheckResourceAttr(dsAddr, "file_name", fileName),
					resource.TestCheckResourceAttr(dsAddr, "info", "ds-by-name"),
				),
			},
		},
	})
}
