// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

package location_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/querycheck"
	"github.com/hashicorp/terraform-plugin-testing/querycheck/queryfilter"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/testhelpers"
)

// TestAccListResource_ProVolumePurchasingLocation_Basic exercises the
// jamfplatform_pro_volume_purchasing_location list resource via the
// `terraform query` workflow. The Pro list endpoint returns the full
// location shape per row, so include_resource = true does not trigger an
// N+1 follow-up GET. Gated on JAMFPLATFORM_VPP_TOKEN to keep CI deterministic
// — provisioning the source resource needs a real Apple-issued service token.
func TestAccListResource_ProVolumePurchasingLocation_Basic(t *testing.T) {
	token := os.Getenv(vppTokenEnvVar)
	if token == "" {
		t.Skipf("%s not set; skipping VPP list resource acceptance test", vppTokenEnvVar)
	}

	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-pro-vpp-list-" + suffix

	resource.Test(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_14_0),
		},
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckVolumePurchasingLocationDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "jamfplatform_pro_volume_purchasing_location" "src" {
						name                     = %q
						service_token            = %q
						service_token_wo_version = 1
					}
				`, name, token),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("jamfplatform_pro_volume_purchasing_location.src", "id"),
				),
			},
			{
				Query: true,
				Config: fmt.Sprintf(`
					provider "jamfplatform" {}

					list "jamfplatform_pro_volume_purchasing_location" "test" {
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
					querycheck.ExpectLength("jamfplatform_pro_volume_purchasing_location.test", 1),
					querycheck.ExpectResourceKnownValues(
						"jamfplatform_pro_volume_purchasing_location.test",
						queryfilter.ByDisplayName(knownvalue.StringExact(name)),
						[]querycheck.KnownValueCheck{
							{Path: tfjsonpath.New("name"), KnownValue: knownvalue.StringExact(name)},
						},
					),
				},
			},
		},
	})
}
