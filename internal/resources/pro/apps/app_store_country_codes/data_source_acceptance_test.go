// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

package app_store_country_codes_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/testhelpers"
)

const codesAddr = "data.jamfplatform_pro_app_store_country_codes.test"

// TestAccDataSource_ProAppStoreCountryCodes_All reads the full country-code list.
func TestAccDataSource_ProAppStoreCountryCodes_All(t *testing.T) {
	testhelpers.AccPreCheck(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
					data "jamfplatform_pro_app_store_country_codes" "test" {}
				`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(codesAddr, "id", "app_store_country_codes"),
					resource.TestCheckResourceAttrSet(codesAddr, "country_codes.#"),
				),
			},
		},
	})
}

// TestAccDataSource_ProAppStoreCountryCodes_Search narrows the list with a substring.
func TestAccDataSource_ProAppStoreCountryCodes_Search(t *testing.T) {
	testhelpers.AccPreCheck(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
					data "jamfplatform_pro_app_store_country_codes" "test" {
						search = "united"
					}
				`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(codesAddr, "country_codes.#"),
				),
			},
		},
	})
}
