// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

package automated_device_enrollment_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/testhelpers"
)

// TestAccDataSource_ProAutomatedDeviceEnrollment_ByID provisions an ADE
// instance via the resource and reads it back through the singular data
// source by ID. Gated on JAMFPLATFORM_ACC_PRO_DEP_TOKEN because the resource Create
// requires a real Apple-issued server token; tokens MUST come from env, never
// committed to fixtures.
func TestAccDataSource_ProAutomatedDeviceEnrollment_ByID(t *testing.T) {
	token := testhelpers.AccEnv(adeTokenEnvVar)
	if token == "" {
		t.Skipf("%s not set; skipping ADE data source acceptance test", adeTokenEnvVar)
	}

	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-pro-ade-ds-id-" + suffix

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckAutomatedDeviceEnrollmentDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "jamfplatform_pro_automated_device_enrollment" "src" {
						name                    = %q
						server_token            = %q
						server_token_wo_version = 1
					}

					data "jamfplatform_pro_automated_device_enrollment" "lookup" {
						id = jamfplatform_pro_automated_device_enrollment.src.id
					}
				`, name, token),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.jamfplatform_pro_automated_device_enrollment.lookup", "name", "jamfplatform_pro_automated_device_enrollment.src", "name"),
					resource.TestCheckResourceAttrSet("data.jamfplatform_pro_automated_device_enrollment.lookup", "token_expiration_date"),
				),
			},
		},
	})
}
