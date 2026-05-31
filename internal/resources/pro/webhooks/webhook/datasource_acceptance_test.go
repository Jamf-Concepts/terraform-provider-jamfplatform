// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

package webhook_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/testhelpers"
)

func TestAccDataSource_ProWebhook_ByID(t *testing.T) {
	testhelpers.AccPreCheck(t)
	name := "tf-acc-pro-webhook-ds-id-" + testhelpers.RunSuffix()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckWebhookDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "jamfplatform_pro_webhook" "src" {
						name                = %q
						url                 = "https://example.com/ds-id"
						event               = "ComputerCheckIn"
						content_type        = "application/json"
						authentication_type = "BASIC"
						username            = "ds-id-user"
						password            = "change-me"
						password_wo_version = 1
					}

					data "jamfplatform_pro_webhook" "lookup" {
						id = jamfplatform_pro_webhook.src.id
					}
				`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.jamfplatform_pro_webhook.lookup", "name", "jamfplatform_pro_webhook.src", "name"),
					resource.TestCheckResourceAttr("data.jamfplatform_pro_webhook.lookup", "event", "ComputerCheckIn"),
					resource.TestCheckResourceAttr("data.jamfplatform_pro_webhook.lookup", "content_type", "application/json"),
					resource.TestCheckResourceAttr("data.jamfplatform_pro_webhook.lookup", "authentication_type", "BASIC"),
					resource.TestCheckResourceAttr("data.jamfplatform_pro_webhook.lookup", "username", "ds-id-user"),
				),
			},
		},
	})
}

func TestAccDataSource_ProWebhook_ByName(t *testing.T) {
	testhelpers.AccPreCheck(t)
	name := "tf-acc-pro-webhook-ds-name-" + testhelpers.RunSuffix()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckWebhookDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "jamfplatform_pro_webhook" "src" {
						name  = %q
						url   = "https://example.com/ds-name"
						event = "MobileDeviceEnrolled"
					}

					data "jamfplatform_pro_webhook" "lookup" {
						name = jamfplatform_pro_webhook.src.name
					}
				`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.jamfplatform_pro_webhook.lookup", "id", "jamfplatform_pro_webhook.src", "id"),
					resource.TestCheckResourceAttr("data.jamfplatform_pro_webhook.lookup", "name", name),
					resource.TestCheckResourceAttr("data.jamfplatform_pro_webhook.lookup", "event", "MobileDeviceEnrolled"),
					resource.TestCheckResourceAttr("data.jamfplatform_pro_webhook.lookup", "authentication_type", "NONE"),
				),
			},
		},
	})
}
