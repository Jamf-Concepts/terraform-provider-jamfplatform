// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

package ibeacon_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/testhelpers"
)

func TestAccDataSource_ProIbeacon_ByID(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-ibeacon-ds-id-" + suffix

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckIbeaconDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "jamfplatform_pro_ibeacon" "src" {
						name  = %q
						uuid  = %q
						major = 7
						minor = 8
					}

					data "jamfplatform_pro_ibeacon" "lookup" {
						id = jamfplatform_pro_ibeacon.src.id
					}
				`, name, acceptanceTestUUID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.jamfplatform_pro_ibeacon.lookup", "name", "jamfplatform_pro_ibeacon.src", "name"),
					resource.TestCheckResourceAttrPair("data.jamfplatform_pro_ibeacon.lookup", "uuid", "jamfplatform_pro_ibeacon.src", "uuid"),
					resource.TestCheckResourceAttr("data.jamfplatform_pro_ibeacon.lookup", "major", "7"),
					resource.TestCheckResourceAttr("data.jamfplatform_pro_ibeacon.lookup", "minor", "8"),
				),
			},
		},
	})
}

func TestAccDataSource_ProIbeacon_ByName(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-ibeacon-ds-name-" + suffix

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckIbeaconDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "jamfplatform_pro_ibeacon" "src" {
						name                    = %q
						uuid                    = %q
						include_any_major_value = true
						include_any_minor_value = true
					}

					data "jamfplatform_pro_ibeacon" "lookup" {
						name = jamfplatform_pro_ibeacon.src.name
					}
				`, name, acceptanceTestUUID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.jamfplatform_pro_ibeacon.lookup", "id", "jamfplatform_pro_ibeacon.src", "id"),
					resource.TestCheckResourceAttr("data.jamfplatform_pro_ibeacon.lookup", "include_any_major_value", "true"),
					resource.TestCheckResourceAttr("data.jamfplatform_pro_ibeacon.lookup", "include_any_minor_value", "true"),
				),
			},
		},
	})
}
