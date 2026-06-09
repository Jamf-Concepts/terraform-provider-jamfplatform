// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

package venafi_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/testhelpers"
)

// TestAccDataSource_ProPkiVenafi reads back a Venafi CA created in the same
// config via the data source.
func TestAccDataSource_ProPkiVenafi(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-pki-venafi-ds-" + suffix

	config := fmt.Sprintf(`
resource "jamfplatform_pro_pki_venafi" "test" {
  name               = %q
  proxy_address      = "proxy.example.com:8443"
  client_id          = "venafi-ds-client"
  revocation_enabled = true
}

data "jamfplatform_pro_pki_venafi" "test" {
  id = jamfplatform_pro_pki_venafi.test.id
}
`, name)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testhelpers.AccPreCheck(t) },
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPkiVenafiDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.jamfplatform_pro_pki_venafi.test", "id", "jamfplatform_pro_pki_venafi.test", "id"),
					resource.TestCheckResourceAttr("data.jamfplatform_pro_pki_venafi.test", "name", name),
					resource.TestCheckResourceAttr("data.jamfplatform_pro_pki_venafi.test", "proxy_address", "proxy.example.com:8443"),
					resource.TestCheckResourceAttr("data.jamfplatform_pro_pki_venafi.test", "client_id", "venafi-ds-client"),
					resource.TestCheckResourceAttr("data.jamfplatform_pro_pki_venafi.test", "revocation_enabled", "true"),
					resource.TestCheckResourceAttrSet("data.jamfplatform_pro_pki_venafi.test", "jamf_public_key"),
				),
			},
		},
	})
}
