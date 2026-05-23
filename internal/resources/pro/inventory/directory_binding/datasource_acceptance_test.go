// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

package directory_binding_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/testhelpers"
)

func TestAccDataSource_ProDirectoryBinding_ByID(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-directory-binding-ds-id-" + suffix

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDirectoryBindingDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "jamfplatform_pro_directory_binding" "src" {
						name     = %q
						type     = "Open Directory"
						domain   = "ldap.example.com"
						username = "ds-id-user"
						password = "change-me"

						open_directory = {
							encrypt_using_ssl      = true
							use_for_authentication = true
						}
					}

					data "jamfplatform_pro_directory_binding" "lookup" {
						id = jamfplatform_pro_directory_binding.src.id
					}
				`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.jamfplatform_pro_directory_binding.lookup", "name", "jamfplatform_pro_directory_binding.src", "name"),
					resource.TestCheckResourceAttr("data.jamfplatform_pro_directory_binding.lookup", "type", "Open Directory"),
					resource.TestCheckResourceAttr("data.jamfplatform_pro_directory_binding.lookup", "open_directory.encrypt_using_ssl", "true"),
					resource.TestCheckResourceAttrSet("data.jamfplatform_pro_directory_binding.lookup", "password_sha256"),
				),
			},
		},
	})
}

func TestAccDataSource_ProDirectoryBinding_ByName(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-directory-binding-ds-name-" + suffix

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDirectoryBindingDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "jamfplatform_pro_directory_binding" "src" {
						name     = %q
						type     = "Centrify"
						domain   = "corp.example.com"
						username = "ds-name-user"
						password = "change-me"

						centrify = {
							zone = "macs"
						}
					}

					data "jamfplatform_pro_directory_binding" "lookup" {
						name = jamfplatform_pro_directory_binding.src.name
					}
				`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.jamfplatform_pro_directory_binding.lookup", "id", "jamfplatform_pro_directory_binding.src", "id"),
					resource.TestCheckResourceAttr("data.jamfplatform_pro_directory_binding.lookup", "name", name),
					resource.TestCheckResourceAttr("data.jamfplatform_pro_directory_binding.lookup", "type", "Centrify"),
					resource.TestCheckResourceAttr("data.jamfplatform_pro_directory_binding.lookup", "centrify.zone", "macs"),
				),
			},
		},
	})
}
