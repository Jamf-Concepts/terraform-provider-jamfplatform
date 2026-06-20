// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

package ldap_server_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/testhelpers"
)

// TestAccDataSource_ProLdapServer_ByIDAndName creates a server, then resolves
// it both by id and by exact display name.
func TestAccDataSource_ProLdapServer_ByIDAndName(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-ldap-ds-" + suffix

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckLdapServerDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "jamfplatform_pro_ldap_server" "test" {
						connection_settings = {
							display_name        = %q
							directory_service   = "Active Directory"
							hostname            = "ldap.acc-ds.example.com"
							authentication_type = "none"
						}
					}

					data "jamfplatform_pro_ldap_server" "by_id" {
						id = jamfplatform_pro_ldap_server.test.id
					}

					data "jamfplatform_pro_ldap_server" "by_name" {
						name = jamfplatform_pro_ldap_server.test.connection_settings.display_name
					}
				`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.jamfplatform_pro_ldap_server.by_id", "id", ldapServerResource, "id"),
					resource.TestCheckResourceAttr("data.jamfplatform_pro_ldap_server.by_id", "connection_settings.display_name", name),
					resource.TestCheckResourceAttr("data.jamfplatform_pro_ldap_server.by_id", "connection_settings.directory_service", "Active Directory"),
					resource.TestCheckResourceAttrPair("data.jamfplatform_pro_ldap_server.by_name", "id", ldapServerResource, "id"),
					resource.TestCheckResourceAttr("data.jamfplatform_pro_ldap_server.by_name", "connection_settings.hostname", "ldap.acc-ds.example.com"),
				),
			},
		},
	})
}
