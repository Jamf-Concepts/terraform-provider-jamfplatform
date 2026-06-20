// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

package gsx_connection_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/testhelpers"
)

// TestAccDataSource_ProGsxConnectionSettings_Basic reads the GSX Connection settings via the
// data source after the resource configures them. Requires real GSX credentials to first
// apply the resource (every write is validated against Apple's GSX service).
func TestAccDataSource_ProGsxConnectionSettings_Basic(t *testing.T) {
	testhelpers.AccPreCheck(t)
	creds := requireGsxCreds(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             checkSingletonRecordStillExists(t),
		Steps: []resource.TestStep{
			{
				Config: gsxConfig(creds, false) + `
data "jamfplatform_pro_gsx_connection_settings" "ds" {
  depends_on = [jamfplatform_pro_gsx_connection_settings.test]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.jamfplatform_pro_gsx_connection_settings.ds", "id", "singleton"),
					resource.TestCheckResourceAttr("data.jamfplatform_pro_gsx_connection_settings.ds", "username", creds.username),
					resource.TestCheckResourceAttr("data.jamfplatform_pro_gsx_connection_settings.ds", "service_account_number", creds.serviceAccount),
					resource.TestCheckResourceAttrSet("data.jamfplatform_pro_gsx_connection_settings.ds", "keystore_expiration_epoch"),
				),
			},
		},
	})
}
