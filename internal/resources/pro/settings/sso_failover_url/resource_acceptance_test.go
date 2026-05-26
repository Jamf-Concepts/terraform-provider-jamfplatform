// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

package sso_failover_url_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/testhelpers"
)

// TestAccResource_ProSsoFailoverURL_Basic exercises Create + a regeneration
// (bumped `regeneration_trigger`) against the live tenant. Destroy is
// state-only so no CheckDestroy is wired.
func TestAccResource_ProSsoFailoverURL_Basic(t *testing.T) {
	testhelpers.AccPreCheck(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
					resource "jamfplatform_pro_sso_failover_url" "test" {
						regeneration_trigger = "v1"
					}
				`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("jamfplatform_pro_sso_failover_url.test", "id", "singleton"),
					resource.TestCheckResourceAttrSet("jamfplatform_pro_sso_failover_url.test", "failover_url"),
					resource.TestCheckResourceAttrSet("jamfplatform_pro_sso_failover_url.test", "generation_time"),
					resource.TestCheckResourceAttrSet("jamfplatform_pro_sso_failover_url.test", "generation_time_utc"),
				),
			},
			{
				Config: `
					resource "jamfplatform_pro_sso_failover_url" "test" {
						regeneration_trigger = "v2"
					}
				`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("jamfplatform_pro_sso_failover_url.test", "regeneration_trigger", "v2"),
					resource.TestCheckResourceAttrSet("jamfplatform_pro_sso_failover_url.test", "failover_url"),
				),
			},
			{
				ResourceName:      "jamfplatform_pro_sso_failover_url.test",
				ImportState:       true,
				ImportStateId:     "singleton",
				ImportStateVerify: false, // regeneration_trigger is user-authored; importer cannot reconstruct it.
			},
		},
	})
}

// TestAccDataSource_ProSsoFailoverURL_Basic exercises the read-only mirror.
func TestAccDataSource_ProSsoFailoverURL_Basic(t *testing.T) {
	testhelpers.AccPreCheck(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
					resource "jamfplatform_pro_sso_failover_url" "src" {
						regeneration_trigger = "v1"
					}
					data "jamfplatform_pro_sso_failover_url" "ds" {
						depends_on = [jamfplatform_pro_sso_failover_url.src]
					}
				`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.jamfplatform_pro_sso_failover_url.ds", "id", "singleton"),
					resource.TestCheckResourceAttrSet("data.jamfplatform_pro_sso_failover_url.ds", "failover_url"),
				),
			},
		},
	})
}
