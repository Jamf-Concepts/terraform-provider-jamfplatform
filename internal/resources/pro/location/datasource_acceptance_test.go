// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

package location_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/testhelpers"
)

// TestAccDataSource_ProVolumePurchasingLocation_ByID provisions a VPP
// location via the resource and reads it back through the singular data
// source by ID. Gated on JAMFPLATFORM_VPP_TOKEN because the resource Create
// requires a real Apple-issued `.vpptoken`; tokens MUST come from env, never
// committed to fixtures.
func TestAccDataSource_ProVolumePurchasingLocation_ByID(t *testing.T) {
	token := os.Getenv(vppTokenEnvVar)
	if token == "" {
		t.Skipf("%s not set; skipping VPP data source acceptance test", vppTokenEnvVar)
	}

	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-pro-vpp-ds-id-" + suffix

	resource.Test(t, resource.TestCase{
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

					data "jamfplatform_pro_volume_purchasing_location" "lookup" {
						id = jamfplatform_pro_volume_purchasing_location.src.id
					}
				`, name, token),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.jamfplatform_pro_volume_purchasing_location.lookup", "name", "jamfplatform_pro_volume_purchasing_location.src", "name"),
					resource.TestCheckResourceAttrSet("data.jamfplatform_pro_volume_purchasing_location.lookup", "token_expiration"),
				),
			},
		},
	})
}
