// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

package smtp_server_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/testhelpers"
)

// TestAccDataSource_ProSmtpServer_Basic reads back the settings the resource just
// applied and confirms the data source mirrors the non-secret fields.
func TestAccDataSource_ProSmtpServer_Basic(t *testing.T) {
	testhelpers.AccPreCheck(t)

	const dsAddr = "data.jamfplatform_pro_smtp_server.lookup"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             checkSingletonRecordStillExists(t),
		Steps: []resource.TestStep{
			{
				Config: `
					resource "jamfplatform_pro_smtp_server" "src" {
						authentication_type = "BASIC"
						sender_settings = {
							email_address = "notifications@example.com"
							display_name  = "DS Example"
						}
						connection_settings = {
							host            = "192.0.2.25"
							port            = 465
							encryption_type = "SSL"
						}
						basic_auth_credentials = {
							username            = "svc@example.com"
							password            = "dummy-password-1"
							password_wo_version = 1
						}
					}

					data "jamfplatform_pro_smtp_server" "lookup" {
						depends_on = [jamfplatform_pro_smtp_server.src]
					}
				`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dsAddr, "id", "singleton"),
					resource.TestCheckResourceAttr(dsAddr, "authentication_type", "BASIC"),
					resource.TestCheckResourceAttr(dsAddr, "sender_settings.display_name", "DS Example"),
					resource.TestCheckResourceAttr(dsAddr, "connection_settings.host", "192.0.2.25"),
					resource.TestCheckResourceAttr(dsAddr, "basic_auth_credentials.username", "svc@example.com"),
				),
			},
		},
	})
}
