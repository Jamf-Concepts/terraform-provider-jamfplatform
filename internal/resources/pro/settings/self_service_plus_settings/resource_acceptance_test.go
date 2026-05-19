// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

package self_service_plus_settings_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/testhelpers"
)

// TestAccResource_ProSelfServicePlusSettings_Basic toggles Self Service Plus on, then
// off, exercising both Update paths against a real tenant. Singleton resources have
// no remote Delete, so CheckDestroy is omitted — the record persists on the tenant
// after Terraform stops managing it.
func TestAccResource_ProSelfServicePlusSettings_Basic(t *testing.T) {
	testhelpers.AccPreCheck(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
					resource "jamfplatform_pro_self_service_plus_settings" "test" {
						enabled = true
					}
				`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("jamfplatform_pro_self_service_plus_settings.test", "id", "singleton"),
					resource.TestCheckResourceAttr("jamfplatform_pro_self_service_plus_settings.test", "enabled", "true"),
				),
			},
			{
				Config: `
					resource "jamfplatform_pro_self_service_plus_settings" "test" {
						enabled = false
					}
				`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("jamfplatform_pro_self_service_plus_settings.test", "enabled", "false"),
				),
			},
			{
				ResourceName:      "jamfplatform_pro_self_service_plus_settings.test",
				ImportState:       true,
				ImportStateId:     "singleton",
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"timeouts",
				},
			},
		},
	})
}

func TestAccDataSource_ProSelfServicePlusSettings_Basic(t *testing.T) {
	testhelpers.AccPreCheck(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
					resource "jamfplatform_pro_self_service_plus_settings" "src" {
						enabled = false
					}

					data "jamfplatform_pro_self_service_plus_settings" "lookup" {
						depends_on = [jamfplatform_pro_self_service_plus_settings.src]
					}
				`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.jamfplatform_pro_self_service_plus_settings.lookup", "id", "singleton"),
					resource.TestCheckResourceAttrPair("data.jamfplatform_pro_self_service_plus_settings.lookup", "enabled", "jamfplatform_pro_self_service_plus_settings.src", "enabled"),
				),
			},
		},
	})
}
