// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

package dns_hostname_mappings_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/testhelpers"
)

// TestAccDataSource_SecurityCloudDNSHostnameMappings_ReadsManagedValues reads back
// mappings the resource wrote, closing the loop between the two — including that an
// omitted address list surfaces as an empty collection rather than null on the read
// side, which is the deliberate difference from the resource model.
func TestAccDataSource_SecurityCloudDNSHostnameMappings_ReadsManagedValues(t *testing.T) {
	testhelpers.AccPreCheckSecurityCloud(t)
	requireEmptyHostnameMappings(t)
	suffix := testhelpers.RunSuffix()
	hostname := "tf-acc-ds-" + suffix + ".example.com"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckHostnameMappingsDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: `
					resource "jamfplatform_security_cloud_dns_hostname_mappings" "test" {
						mappings = [
							{
								hostname              = "` + hostname + `"
								ipv4_addresses        = ["10.0.0.1"]
								connect_to_ztna       = true
								connect_to_secure_dns = false
							},
						]
					}

					data "jamfplatform_security_cloud_dns_hostname_mappings" "test" {
						depends_on = [jamfplatform_security_cloud_dns_hostname_mappings.test]
					}
				`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.jamfplatform_security_cloud_dns_hostname_mappings.test", "id", helpers.SingletonID),
					resource.TestCheckResourceAttr("data.jamfplatform_security_cloud_dns_hostname_mappings.test", "mappings.#", "1"),
					resource.TestCheckResourceAttr("data.jamfplatform_security_cloud_dns_hostname_mappings.test", "mappings.0.hostname", hostname),
					resource.TestCheckResourceAttr("data.jamfplatform_security_cloud_dns_hostname_mappings.test", "mappings.0.ipv4_addresses.#", "1"),
					resource.TestCheckResourceAttr("data.jamfplatform_security_cloud_dns_hostname_mappings.test", "mappings.0.ipv6_addresses.#", "0"),
					resource.TestCheckResourceAttr("data.jamfplatform_security_cloud_dns_hostname_mappings.test", "mappings.0.connect_to_ztna", "true"),
					resource.TestCheckResourceAttr("data.jamfplatform_security_cloud_dns_hostname_mappings.test", "mappings.0.connect_to_secure_dns", "false"),
				),
			},
		},
	})
}

// TestAccDataSource_SecurityCloudDNSHostnameMappings_EmptyIsNotAnError pins the
// deliberate difference from the sibling search domain data source. An empty mapping
// set is an ordinary 200 that a for expression handles on its own, so there is nothing
// to fail on — whereas an unset search domain is a 404 and does error.
func TestAccDataSource_SecurityCloudDNSHostnameMappings_EmptyIsNotAnError(t *testing.T) {
	testhelpers.AccPreCheckSecurityCloud(t)
	requireEmptyHostnameMappings(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `data "jamfplatform_security_cloud_dns_hostname_mappings" "test" {}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.jamfplatform_security_cloud_dns_hostname_mappings.test", "id", helpers.SingletonID),
					resource.TestCheckResourceAttr("data.jamfplatform_security_cloud_dns_hostname_mappings.test", "mappings.#", "0"),
				),
			},
		},
	})
}
