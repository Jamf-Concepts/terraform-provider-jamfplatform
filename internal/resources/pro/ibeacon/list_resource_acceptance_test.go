// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

package ibeacon_test

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

// TestAccListResource_ProIbeacon_Basic exercises the jamfplatform_pro_ibeacon
// list resource via the `terraform query` workflow.
func TestAccListResource_ProIbeacon_Basic(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-ibeacon-list-" + suffix

	resource.Test(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_14_0),
		},
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckIbeaconDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "jamfplatform_pro_ibeacon" "src" {
						name  = %q
						uuid  = %q
						major = 99
						minor = 88
					}
				`, name, acceptanceTestUUID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("jamfplatform_pro_ibeacon.src", "id"),
				),
			},
			{
				Query: true,
				Config: fmt.Sprintf(`
					provider "jamfplatform" {}

					list "jamfplatform_pro_ibeacon" "test" {
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
					querycheck.ExpectLength("jamfplatform_pro_ibeacon.test", 1),
					querycheck.ExpectResourceKnownValues(
						"jamfplatform_pro_ibeacon.test",
						queryfilter.ByDisplayName(knownvalue.StringExact(name)),
						[]querycheck.KnownValueCheck{
							{Path: tfjsonpath.New("name"), KnownValue: knownvalue.StringExact(name)},
							{Path: tfjsonpath.New("uuid"), KnownValue: knownvalue.StringExact(acceptanceTestUUID)},
							{Path: tfjsonpath.New("major"), KnownValue: knownvalue.Int64Exact(99)},
							{Path: tfjsonpath.New("minor"), KnownValue: knownvalue.Int64Exact(88)},
							{Path: tfjsonpath.New("include_any_major_value"), KnownValue: knownvalue.Bool(false)},
							{Path: tfjsonpath.New("include_any_minor_value"), KnownValue: knownvalue.Bool(false)},
						},
					),
				},
			},
		},
	})
}
