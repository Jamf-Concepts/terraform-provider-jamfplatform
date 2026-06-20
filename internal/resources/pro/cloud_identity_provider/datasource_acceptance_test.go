// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

// Tests in this file verify the jamfplatform_pro_cloud_identity_provider (singular) and
// jamfplatform_pro_cloud_identity_providers (plural) data sources. A Microsoft Entra ID Cloud
// Identity Provider is created as the backing resource (no env gate required —
// the server accepts the placeholder consent code) and then looked up via both
// data sources.

package cloud_identity_provider_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/testhelpers"
)

// azureBackingResource returns the HCL for a minimal Entra ID Cloud Identity
// Provider used as a backing resource for data-source tests.
func azureBackingResource(displayName string) string {
	return fmt.Sprintf(`
resource "jamfplatform_pro_cloud_identity_provider" "src" {
  display_name  = %q
  provider_name = "ENTRA_ID"

  entra_id = {
    tenant_id = "d5749c84-5cc5-4691-a187-4545c02ff915"
  }
}
`, displayName)
}

// TestAccDataSource_ProCloudIdp_ByID creates an Azure CIP and reads it back via
// the jamfplatform_pro_cloud_identity_provider singular data source using the resource ID.
func TestAccDataSource_ProCloudIdp_ByID(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	displayName := "tf-acc-cip-ds-id-" + suffix

	config := azureBackingResource(displayName) + `
data "jamfplatform_pro_cloud_identity_provider" "lookup" {
  id = jamfplatform_pro_cloud_identity_provider.src.id
}
`

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckCloudIdentityProviderDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"data.jamfplatform_pro_cloud_identity_provider.lookup", "id",
						"jamfplatform_pro_cloud_identity_provider.src", "id",
					),
					resource.TestCheckResourceAttrPair(
						"data.jamfplatform_pro_cloud_identity_provider.lookup", "display_name",
						"jamfplatform_pro_cloud_identity_provider.src", "display_name",
					),
					resource.TestCheckResourceAttr("data.jamfplatform_pro_cloud_identity_provider.lookup", "provider_name", "ENTRA_ID"),
					resource.TestCheckResourceAttrSet("data.jamfplatform_pro_cloud_identity_provider.lookup", "enabled"),
					resource.TestCheckResourceAttrSet("data.jamfplatform_pro_cloud_identity_provider.lookup", "provider_description"),
				),
			},
		},
	})
}

// TestAccDataSource_ProCloudIdp_ByDisplayName creates an Azure CIP and reads it
// back via the jamfplatform_pro_cloud_identity_provider data source using the display_name.
func TestAccDataSource_ProCloudIdp_ByDisplayName(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	displayName := "tf-acc-cip-ds-name-" + suffix

	config := azureBackingResource(displayName) + `
data "jamfplatform_pro_cloud_identity_provider" "lookup" {
  display_name = jamfplatform_pro_cloud_identity_provider.src.display_name
}
`

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckCloudIdentityProviderDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"data.jamfplatform_pro_cloud_identity_provider.lookup", "id",
						"jamfplatform_pro_cloud_identity_provider.src", "id",
					),
					resource.TestCheckResourceAttr("data.jamfplatform_pro_cloud_identity_provider.lookup", "display_name", displayName),
					resource.TestCheckResourceAttr("data.jamfplatform_pro_cloud_identity_provider.lookup", "provider_name", "ENTRA_ID"),
				),
			},
		},
	})
}

// TestAccDataSource_ProCloudIdps_List creates an Azure CIP and verifies the
// jamfplatform_pro_cloud_identity_providers plural data source includes the created provider's
// ID in its cloud_identity_providers list.
func TestAccDataSource_ProCloudIdps_List(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	displayName := "tf-acc-cip-ds-plural-" + suffix

	config := azureBackingResource(displayName) + `
data "jamfplatform_pro_cloud_identity_providers" "all" {
  depends_on = [jamfplatform_pro_cloud_identity_provider.src]
}
`

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckCloudIdentityProviderDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					// The created resource's ID must appear somewhere in the list.
					resource.TestCheckTypeSetElemNestedAttrs(
						"data.jamfplatform_pro_cloud_identity_providers.all",
						"cloud_identity_providers.*",
						map[string]string{
							"display_name": displayName,
						},
					),
				),
			},
		},
	})
}
