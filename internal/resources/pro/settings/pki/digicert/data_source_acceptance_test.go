// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

package digicert_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/testhelpers"
)

// TestAccDataSource_ProPkiDigicert reads a DigiCert TLM integration via the data
// source after the resource creates it (committed dummy .p12 — no real DigiCert
// CA needed at create). The Computed client_certificate_details block is asserted
// populated.
func TestAccDataSource_ProPkiDigicert(t *testing.T) {
	testhelpers.AccPreCheck(t)
	b64 := dummyP12Base64(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDigicertDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: digicertConfig("tf-acc-digicert-ds", "one.digicert.com", false, b64, 1) + `
data "jamfplatform_pro_pki_digicert" "ds" {
  id = jamfplatform_pro_pki_digicert.test.id
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.jamfplatform_pro_pki_digicert.ds", "display_name", "tf-acc-digicert-ds"),
					resource.TestCheckResourceAttr("data.jamfplatform_pro_pki_digicert.ds", "host_name", "one.digicert.com"),
					resource.TestCheckResourceAttr("data.jamfplatform_pro_pki_digicert.ds", "revocation_enabled", "false"),
					resource.TestCheckResourceAttrSet("data.jamfplatform_pro_pki_digicert.ds", "client_certificate_details.serial_number"),
					resource.TestCheckResourceAttr("data.jamfplatform_pro_pki_digicert.ds", "client_certificate_details.subject", "CN=pki-dummy"),
				),
			},
		},
	})
}
