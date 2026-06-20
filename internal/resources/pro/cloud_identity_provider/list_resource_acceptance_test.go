// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

// Tests in this file exercise the jamfplatform_pro_cloud_identity_provider list
// resource via the `terraform query` workflow. A Microsoft Entra ID Cloud
// Identity Provider is created as the backing resource (no env gate required)
// because the list endpoint (/v1/cloud-idp) returns the full
// CloudIDPCommonResponse summary — no follow-up GET is needed for
// include_resource=true. The test pins the name_substring filter client-side
// behaviour and the include_resource path.

package cloud_identity_provider_test

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

// TestAccListResource_ProCloudIdentityProvider_Basic exercises the
// jamfplatform_pro_cloud_identity_provider list resource via the `terraform
// query` workflow. Step 1 creates an Azure CIP; Step 2 queries the list
// resource with a name_substring filter to confirm the created provider appears
// in the results.
func TestAccListResource_ProCloudIdentityProvider_Basic(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	displayName := "tf-acc-cip-list-" + suffix

	resource.Test(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_14_0),
		},
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckCloudIdentityProviderDestroy(t),
		Steps: []resource.TestStep{
			// Step 1: Create the backing Entra ID CIP.
			{
				Config: fmt.Sprintf(`
resource "jamfplatform_pro_cloud_identity_provider" "src" {
  display_name  = %q
  provider_name = "ENTRA_ID"

  entra_id = {
    tenant_id = "d5749c84-5cc5-4691-a187-4545c02ff915"
  }
}
`, displayName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("jamfplatform_pro_cloud_identity_provider.src", "id"),
				),
			},
			// Step 2: Query the list resource with a name_substring filter.
			{
				Query: true,
				Config: fmt.Sprintf(`
provider "jamfplatform" {}

list "jamfplatform_pro_cloud_identity_provider" "test" {
  provider         = jamfplatform
  include_resource = true

  config {
    filter = {
      name_substring = %q
    }
  }
}
`, displayName),
				QueryResultChecks: []querycheck.QueryResultCheck{
					querycheck.ExpectLength("jamfplatform_pro_cloud_identity_provider.test", 1),
					querycheck.ExpectResourceKnownValues(
						"jamfplatform_pro_cloud_identity_provider.test",
						queryfilter.ByDisplayName(knownvalue.StringExact(displayName)),
						[]querycheck.KnownValueCheck{
							{Path: tfjsonpath.New("display_name"), KnownValue: knownvalue.StringExact(displayName)},
							{Path: tfjsonpath.New("provider_name"), KnownValue: knownvalue.StringExact("ENTRA_ID")},
						},
					),
				},
			},
		},
	})
}

// TestAccListResource_ProCloudIdentityProvider_Filter verifies that the
// name_substring filter excludes providers that do not match. A CIP is created
// with a unique name; a query with a non-matching substring must return zero
// results.
func TestAccListResource_ProCloudIdentityProvider_Filter(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	displayName := "tf-acc-cip-list-filter-" + suffix

	resource.Test(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_14_0),
		},
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckCloudIdentityProviderDestroy(t),
		Steps: []resource.TestStep{
			// Step 1: Create the backing Entra ID CIP.
			{
				Config: fmt.Sprintf(`
resource "jamfplatform_pro_cloud_identity_provider" "src" {
  display_name  = %q
  provider_name = "ENTRA_ID"

  entra_id = {
    tenant_id = "d5749c84-5cc5-4691-a187-4545c02ff915"
  }
}
`, displayName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("jamfplatform_pro_cloud_identity_provider.src", "id"),
				),
			},
			// Step 2: Query with a non-matching substring — expect zero results.
			{
				Query: true,
				Config: `
provider "jamfplatform" {}

list "jamfplatform_pro_cloud_identity_provider" "test" {
  provider         = jamfplatform
  include_resource = false

  config {
    filter = {
      name_substring = "this-substring-will-never-match-any-real-record"
    }
  }
}
`,
				QueryResultChecks: []querycheck.QueryResultCheck{
					querycheck.ExpectLength("jamfplatform_pro_cloud_identity_provider.test", 0),
				},
			},
		},
	})
}
