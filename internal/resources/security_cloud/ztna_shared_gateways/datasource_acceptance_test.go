// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

package ztna_shared_gateways_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/testhelpers"
)

// TestAccDataSource_SecurityCloudZtnaSharedGateways_ReadsCatalogue reads the
// Jamf-managed catalogue and asserts it is non-empty and shaped as expected.
//
// It asserts presence rather than specific entries. The catalogue is Jamf's, not
// the tenant's — its contents change when Jamf adds a region, and pinning
// individual IDs would make the test a hostage to that. The fixture helper already
// skips when the tenant cannot read it at all, so an empty result here would mean
// the read succeeded and returned nothing, which is worth failing on.
func TestAccDataSource_SecurityCloudZtnaSharedGateways_ReadsCatalogue(t *testing.T) {
	testhelpers.AccPreCheckSecurityCloud(t)
	testhelpers.RequireSecurityCloudSharedGatewayIDs(t, 1)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `data "jamfplatform_security_cloud_ztna_shared_gateways" "all" {}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.jamfplatform_security_cloud_ztna_shared_gateways.all", "id", "ztna_shared_gateways"),
					resource.TestCheckResourceAttrSet("data.jamfplatform_security_cloud_ztna_shared_gateways.all", "shared_gateways.#"),
					resource.TestCheckResourceAttrSet("data.jamfplatform_security_cloud_ztna_shared_gateways.all", "shared_gateways.0.id"),
					resource.TestCheckResourceAttrSet("data.jamfplatform_security_cloud_ztna_shared_gateways.all", "shared_gateways.0.name"),
				),
			},
		},
	})
}

// TestAccDataSource_SecurityCloudZtnaSharedGateways_ResolvesDNSZoneGateway is the
// integration this data source exists for: it closes the loop from the catalogue
// to a custom DNS zone's name server, which otherwise needs a hard-coded
// four-character ID.
func TestAccDataSource_SecurityCloudZtnaSharedGateways_ResolvesDNSZoneGateway(t *testing.T) {
	testhelpers.AccPreCheckSecurityCloud(t)
	testhelpers.RequireSecurityCloudSharedGatewayIDs(t, 1)
	suffix := testhelpers.RunSuffix()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
					data "jamfplatform_security_cloud_ztna_shared_gateways" "all" {}

					locals {
						nearest = one([
							for gateway in data.jamfplatform_security_cloud_ztna_shared_gateways.all.shared_gateways :
							gateway.id if gateway.name == "Nearest Data Center"
						])
					}

					resource "jamfplatform_security_cloud_dns_zone" "test" {
						name    = "tf-acc-jsc-shared-gw-` + suffix + `"
						domains = ["tf-acc-shared-` + suffix + `.example.com"]

						name_servers = [
							{
								ip         = "203.0.113.53"
								gateway_id = local.nearest
							},
						]
					}
				`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("jamfplatform_security_cloud_dns_zone.test", "id"),
					resource.TestCheckResourceAttrSet("jamfplatform_security_cloud_dns_zone.test", "name_servers.0.gateway_id"),
				),
			},
		},
	})
}
