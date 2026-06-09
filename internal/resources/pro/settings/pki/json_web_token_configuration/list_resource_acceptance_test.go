// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

package json_web_token_configuration_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/querycheck"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/testhelpers"
)

// TestAccListResource_ProPkiJSONWebTokenConfiguration_Basic exercises the list
// resource via the `terraform query` workflow, chained off the single managed
// configuration (max one per tenant). The client-side name_substring filter
// narrows to the record created in step 1.
func TestAccListResource_ProPkiJSONWebTokenConfiguration_Basic(t *testing.T) {
	testhelpers.AccPreCheck(t)
	name := "tf-acc-pro-jwt-list-" + testhelpers.RunSuffix()

	resource.Test(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_14_0),
		},
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckJSONWebTokenConfigurationDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "jamfplatform_pro_pki_json_web_token_configuration" "src" {
						name                      = %q
						encryption_key_wo         = %q
						encryption_key_wo_version = 1
						token_expiry              = 15
					}
				`, name, jwtAccKeyV1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("jamfplatform_pro_pki_json_web_token_configuration.src", "id"),
				),
			},
			{
				Query: true,
				Config: fmt.Sprintf(`
					provider "jamfplatform" {}

					list "jamfplatform_pro_pki_json_web_token_configuration" "test" {
						provider = jamfplatform

						config {
							filter = {
								name_substring = %q
							}
						}
					}
				`, name),
				QueryResultChecks: []querycheck.QueryResultCheck{
					// Identity-only list: the client-side name_substring filter
					// narrows to the single record created above.
					querycheck.ExpectLength("jamfplatform_pro_pki_json_web_token_configuration.test", 1),
				},
			},
		},
	})
}
