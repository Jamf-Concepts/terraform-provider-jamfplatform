// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

package ztna_predefined_apps_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/testhelpers"
)

// TestAccDataSource_SecurityCloudZtnaPredefinedApps_ReadsCatalogue reads the
// Jamf-curated catalogue and asserts it is non-empty and shaped as expected.
//
// It asserts presence rather than specific entries. The catalogue is Jamf's, not
// the tenant's — Jamf adds and revises templates, and pinning a template's ID or
// its hostname list would make the test a hostage to that. The fixture helper
// already skips when the tenant cannot read it at all, so an empty result here
// would mean the read succeeded and returned nothing, which is worth failing on.
func TestAccDataSource_SecurityCloudZtnaPredefinedApps_ReadsCatalogue(t *testing.T) {
	testhelpers.AccPreCheckSecurityCloud(t)
	testhelpers.RequireSecurityCloudPredefinedApps(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `data "jamfplatform_security_cloud_ztna_predefined_apps" "all" {}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.jamfplatform_security_cloud_ztna_predefined_apps.all", "id", "ztna_predefined_apps"),
					resource.TestCheckResourceAttrSet("data.jamfplatform_security_cloud_ztna_predefined_apps.all", "predefined_apps.#"),
					resource.TestCheckResourceAttrSet("data.jamfplatform_security_cloud_ztna_predefined_apps.all", "predefined_apps.0.id"),
					resource.TestCheckResourceAttrSet("data.jamfplatform_security_cloud_ztna_predefined_apps.all", "predefined_apps.0.name"),
					resource.TestCheckResourceAttrSet("data.jamfplatform_security_cloud_ztna_predefined_apps.all", "predefined_apps.0.hostnames.#"),
				),
			},
		},
	})
}

// TestAccDataSource_SecurityCloudZtnaPredefinedApps_CarriesHostnames pins the half
// of a template that makes it reviewable. A template bundles hostnames an app
// inherits wholesale, so a catalogue that read back with empty hostname lists would
// look healthy while being useless — the count assertion above cannot tell the
// difference between "no hostnames" and "not decoded".
func TestAccDataSource_SecurityCloudZtnaPredefinedApps_CarriesHostnames(t *testing.T) {
	testhelpers.AccPreCheckSecurityCloud(t)
	apps := testhelpers.RequireSecurityCloudPredefinedApps(t)

	withHostnames := ""
	for _, app := range apps {
		if len(app.Hostnames) > 0 {
			withHostnames = app.ID
			break
		}
	}
	if withHostnames == "" {
		t.Skip("Skipping: no predefined app on this tenant carries hostnames; nothing to assert")
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
					data "jamfplatform_security_cloud_ztna_predefined_apps" "all" {}

					locals {
						matched = one([
							for app in data.jamfplatform_security_cloud_ztna_predefined_apps.all.predefined_apps :
							app if app.id == "` + withHostnames + `"
						])
					}

					output "matched_hostname_count" {
						value = length(local.matched.hostnames) > 0
					}
				`,
				Check: resource.TestCheckOutput("matched_hostname_count", "true"),
			},
		},
	})
}
