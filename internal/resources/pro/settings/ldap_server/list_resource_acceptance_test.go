// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

package ldap_server_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/querycheck"
	"github.com/hashicorp/terraform-plugin-testing/querycheck/queryfilter"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/testhelpers"
)

// TestAccListResource_ProLdapServer_Basic exercises the
// jamfplatform_pro_ldap_server list resource via the `terraform query`
// workflow. The classic /ldapservers list returns id+name only; with
// include_resource=true the list resource follows up with a singular GET per
// item to populate the full record — this pins the N+1 path end-to-end.
func TestAccListResource_ProLdapServer_Basic(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-ldap-list-" + suffix

	resource.Test(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_14_0),
		},
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckLdapServerDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "jamfplatform_pro_ldap_server" "src" {
						connection_settings = {
							display_name        = %q
							directory_service   = "Active Directory"
							hostname            = "ldap.acc-list.example.com"
							authentication_type = "none"
						}
					}
				`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("jamfplatform_pro_ldap_server.src", "id"),
				),
			},
			{
				Query: true,
				Config: fmt.Sprintf(`
					provider "jamfplatform" {}

					list "jamfplatform_pro_ldap_server" "test" {
						provider         = jamfplatform
						include_resource = true

						config {
							filter = {
								name_substring = %q
							}
						}
					}
				`, name),
				QueryResultChecks: []querycheck.QueryResultCheck{
					querycheck.ExpectLength("jamfplatform_pro_ldap_server.test", 1),
					querycheck.ExpectResourceKnownValues(
						"jamfplatform_pro_ldap_server.test",
						queryfilter.ByDisplayName(knownvalue.StringExact(name)),
						[]querycheck.KnownValueCheck{
							{Path: tfjsonpath.New("connection_settings").AtMapKey("display_name"), KnownValue: knownvalue.StringExact(name)},
							{Path: tfjsonpath.New("connection_settings").AtMapKey("directory_service"), KnownValue: knownvalue.StringExact("Active Directory")},
						},
					),
				},
			},
		},
	})
}
