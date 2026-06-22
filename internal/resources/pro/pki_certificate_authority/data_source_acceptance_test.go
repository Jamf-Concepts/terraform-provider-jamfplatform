// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

package pki_certificate_authority_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/testhelpers"
)

// TestAccDataSource_ProPKICertificateAuthority_Active reads the active Certificate
// Authority (the Jamf Pro built-in CA on most tenants). No fixture required — every tenant
// has an active CA.
func TestAccDataSource_ProPKICertificateAuthority_Active(t *testing.T) {
	testhelpers.AccPreCheck(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
data "jamfplatform_pro_pki_certificate_authority" "active" {}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.jamfplatform_pro_pki_certificate_authority.active", "id", "active"),
					resource.TestCheckResourceAttrSet("data.jamfplatform_pro_pki_certificate_authority.active", "subject_x500_principal"),
					resource.TestCheckResourceAttrSet("data.jamfplatform_pro_pki_certificate_authority.active", "issuer_x500_principal"),
					resource.TestCheckResourceAttrSet("data.jamfplatform_pro_pki_certificate_authority.active", "serial_number"),
					resource.TestCheckResourceAttrSet("data.jamfplatform_pro_pki_certificate_authority.active", "not_after"),
					resource.TestCheckResourceAttrSet("data.jamfplatform_pro_pki_certificate_authority.active", "sha256_fingerprint"),
					resource.TestCheckResourceAttrSet("data.jamfplatform_pro_pki_certificate_authority.active", "pem"),
				),
			},
		},
	})
}
