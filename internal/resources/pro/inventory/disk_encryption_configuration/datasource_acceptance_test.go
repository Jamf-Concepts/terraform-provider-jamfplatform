// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

package disk_encryption_configuration_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/testhelpers"
)

func TestAccDataSource_ProDiskEncryptionConfiguration_ByID(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-disk-encryption-ds-id-" + suffix

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDiskEncryptionConfigurationDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "jamfplatform_pro_disk_encryption_configuration" "src" {
						name                     = %q
						key_type                 = "Individual"
						file_vault_enabled_users = "Current or Next User"
					}

					data "jamfplatform_pro_disk_encryption_configuration" "lookup" {
						id = jamfplatform_pro_disk_encryption_configuration.src.id
					}
				`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.jamfplatform_pro_disk_encryption_configuration.lookup", "name", "jamfplatform_pro_disk_encryption_configuration.src", "name"),
					resource.TestCheckResourceAttr("data.jamfplatform_pro_disk_encryption_configuration.lookup", "key_type", "Individual"),
				),
			},
		},
	})
}

// TestAccDataSource_ProDiskEncryptionConfiguration_ByName uses the
// classic /diskencryptionconfigurations/name/{name} endpoint directly.
// Probed 2026-05-23 and confirmed working (unlike
// /directorybindings/name/{name} which returns 500); the SDK's
// GetDiskEncryptionConfigurationByName is wired in straight, no list+ID
// match workaround needed.
func TestAccDataSource_ProDiskEncryptionConfiguration_ByName(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-disk-encryption-ds-name-" + suffix

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDiskEncryptionConfigurationDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "jamfplatform_pro_disk_encryption_configuration" "src" {
						name                     = %q
						key_type                 = "Individual"
						file_vault_enabled_users = "Management Account"
					}

					data "jamfplatform_pro_disk_encryption_configuration" "lookup" {
						name = jamfplatform_pro_disk_encryption_configuration.src.name
					}
				`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.jamfplatform_pro_disk_encryption_configuration.lookup", "id", "jamfplatform_pro_disk_encryption_configuration.src", "id"),
					resource.TestCheckResourceAttr("data.jamfplatform_pro_disk_encryption_configuration.lookup", "name", name),
					resource.TestCheckResourceAttr("data.jamfplatform_pro_disk_encryption_configuration.lookup", "key_type", "Individual"),
					resource.TestCheckResourceAttr("data.jamfplatform_pro_disk_encryption_configuration.lookup", "file_vault_enabled_users", "Management Account"),
				),
			},
		},
	})
}
