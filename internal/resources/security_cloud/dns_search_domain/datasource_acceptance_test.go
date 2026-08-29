// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

package dns_search_domain_test

import (
	"context"
	"regexp"
	"testing"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/securitycloud"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/testhelpers"
)

// TestAccDataSource_SecurityCloudDNSSearchDomain_ReadsManagedValue reads back a
// search domain the resource wrote, closing the loop between the two.
func TestAccDataSource_SecurityCloudDNSSearchDomain_ReadsManagedValue(t *testing.T) {
	testhelpers.AccPreCheckSecurityCloud(t)
	clearSearchDomain(t)
	suffix := testhelpers.RunSuffix()
	domain := "tf-acc-ds-" + suffix + ".example.com"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckSearchDomainDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: `
					resource "jamfplatform_security_cloud_dns_search_domain" "test" {
						domain_name = "` + domain + `"
					}

					data "jamfplatform_security_cloud_dns_search_domain" "test" {
						depends_on = [jamfplatform_security_cloud_dns_search_domain.test]
					}
				`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.jamfplatform_security_cloud_dns_search_domain.test", "id", helpers.SingletonID),
					resource.TestCheckResourceAttr("data.jamfplatform_security_cloud_dns_search_domain.test", "domain_name", domain),
				),
			},
		},
	})
}

// TestAccDataSource_SecurityCloudDNSSearchDomain_ErrorsWhenUnset pins that an
// unconfigured tenant is an error rather than an empty string. A data source that
// quietly produced "" would feed that into whatever referenced it, and the endpoint's
// 404 is unambiguous enough that there is no reason to guess.
func TestAccDataSource_SecurityCloudDNSSearchDomain_ErrorsWhenUnset(t *testing.T) {
	testhelpers.AccPreCheckSecurityCloud(t)

	c := securitycloud.New(testhelpers.NewAcceptanceClient(t))
	if err := c.ClearDnsSearchDomainV1(context.Background()); err != nil {
		t.Fatalf("cannot clear the tenant's search domain: %v", err)
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      `data "jamfplatform_security_cloud_dns_search_domain" "test" {}`,
				ExpectError: regexp.MustCompile(`No Jamf Security Cloud search domain configured`),
			},
		},
	})
}
