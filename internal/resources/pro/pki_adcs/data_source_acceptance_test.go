// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

package pki_adcs_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/testhelpers"
)

// TestAccDataSource_ProPkiAdcs_Inbound reads an INBOUND AD CS integration via the
// data source after the resource creates it (committed dummy certs — no real AD CS
// connector needed at create). Both Computed *_certificate_details blocks and the
// derived connector_mode are asserted populated.
func TestAccDataSource_ProPkiAdcs_Inbound(t *testing.T) {
	testhelpers.AccPreCheck(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckAdcsDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: inboundConfig("tf-acc-adcs-ds", false, 1, 1) + `
data "jamfplatform_pro_pki_adcs" "ds" {
  id = jamfplatform_pro_pki_adcs.test.id
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.jamfplatform_pro_pki_adcs.ds", "connector_mode", "INBOUND"),
					resource.TestCheckResourceAttr("data.jamfplatform_pro_pki_adcs.ds", "display_name", "tf-acc-adcs-ds"),
					resource.TestCheckResourceAttr("data.jamfplatform_pro_pki_adcs.ds", "adcs_url", "connector.example.com"),
					resource.TestCheckResourceAttrSet("data.jamfplatform_pro_pki_adcs.ds", "server_certificate_details.serial_number"),
					resource.TestCheckResourceAttrSet("data.jamfplatform_pro_pki_adcs.ds", "client_certificate_details.serial_number"),
				),
			},
		},
	})
}
