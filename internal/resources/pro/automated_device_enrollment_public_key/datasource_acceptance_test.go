// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

package automated_device_enrollment_public_key_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/testhelpers"
)

// TestAccDataSource_ProAutomatedDeviceEnrollmentPublicKey_Singleton verifies
// the tenant's ADE public key data source returns a non-empty base64 body.
// The endpoint is tenant-wide and read-only, so no JAMFPLATFORM_ADE_TOKEN
// gating is required — only the standard AccPreCheck for tenant credentials.
func TestAccDataSource_ProAutomatedDeviceEnrollmentPublicKey_Singleton(t *testing.T) {
	testhelpers.AccPreCheck(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
					data "jamfplatform_pro_automated_device_enrollment_public_key" "test" {}
				`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.jamfplatform_pro_automated_device_enrollment_public_key.test", "id", "singleton"),
					resource.TestCheckResourceAttrSet("data.jamfplatform_pro_automated_device_enrollment_public_key.test", "public_key"),
				),
			},
		},
	})
}
