// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

package json_web_token_configuration_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/testhelpers"
)

// TestAccDataSource_ProPkiJSONWebTokenConfiguration exercises both lookup modes
// (by id and by exact name) chained off the single managed configuration —
// the server allows only one per tenant, so both data sources read the same
// record in one step.
func TestAccDataSource_ProPkiJSONWebTokenConfiguration(t *testing.T) {
	testhelpers.AccPreCheck(t)
	name := "tf-acc-pro-jwt-ds-" + testhelpers.RunSuffix()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckJSONWebTokenConfigurationDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "jamfplatform_pro_pki_json_web_token_configuration" "src" {
						name                      = %q
						encryption_key_wo         = %q
						encryption_key_wo_version = 1
						token_expiry              = 45
					}

					data "jamfplatform_pro_pki_json_web_token_configuration" "by_id" {
						id = jamfplatform_pro_pki_json_web_token_configuration.src.id
					}

					data "jamfplatform_pro_pki_json_web_token_configuration" "by_name" {
						name = jamfplatform_pro_pki_json_web_token_configuration.src.name
					}
				`, name, jwtAccKeyV1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.jamfplatform_pro_pki_json_web_token_configuration.by_id", "name", "jamfplatform_pro_pki_json_web_token_configuration.src", "name"),
					resource.TestCheckResourceAttr("data.jamfplatform_pro_pki_json_web_token_configuration.by_id", "token_expiry", "45"),
					resource.TestCheckResourceAttr("data.jamfplatform_pro_pki_json_web_token_configuration.by_id", "enabled", "true"),
					resource.TestCheckResourceAttrPair("data.jamfplatform_pro_pki_json_web_token_configuration.by_name", "id", "jamfplatform_pro_pki_json_web_token_configuration.src", "id"),
					resource.TestCheckResourceAttr("data.jamfplatform_pro_pki_json_web_token_configuration.by_name", "name", name),
					resource.TestCheckResourceAttr("data.jamfplatform_pro_pki_json_web_token_configuration.by_name", "token_expiry", "45"),
				),
			},
		},
	})
}
