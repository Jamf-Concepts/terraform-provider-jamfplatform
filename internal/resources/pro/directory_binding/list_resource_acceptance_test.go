// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

package directory_binding_test

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

// TestAccListResource_ProDirectoryBinding_Basic exercises the
// jamfplatform_pro_directory_binding list resource via the
// `terraform query` workflow. The classic /directorybindings list
// endpoint returns id+name only; with include_resource=true the list
// resource follows up with a singular GET per item to populate the full
// record. This test pins the N+1 path end-to-end.
func TestAccListResource_ProDirectoryBinding_Basic(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-directory-binding-list-" + suffix

	resource.Test(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_14_0),
		},
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDirectoryBindingDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "jamfplatform_pro_directory_binding" "src" {
						name     = %q
						priority = 5
						type     = "Centrify"
						domain   = "corp.example.com"
						username = "list-user"
						password = "change-me"

						centrify = {
							zone = "macs"
						}
					}
				`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("jamfplatform_pro_directory_binding.src", "id"),
				),
			},
			{
				Query: true,
				Config: fmt.Sprintf(`
					provider "jamfplatform" {}

					list "jamfplatform_pro_directory_binding" "test" {
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
					querycheck.ExpectLength("jamfplatform_pro_directory_binding.test", 1),
					querycheck.ExpectResourceKnownValues(
						"jamfplatform_pro_directory_binding.test",
						queryfilter.ByDisplayName(knownvalue.StringExact(name)),
						[]querycheck.KnownValueCheck{
							{Path: tfjsonpath.New("name"), KnownValue: knownvalue.StringExact(name)},
							{Path: tfjsonpath.New("type"), KnownValue: knownvalue.StringExact("Centrify")},
							{Path: tfjsonpath.New("centrify").AtMapKey("zone"), KnownValue: knownvalue.StringExact("macs")},
						},
					),
				},
			},
		},
	})
}
