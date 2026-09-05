// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

// Tests in this file exercise the jamfplatform_pro_cloud_identity_provider list
// resource via the `terraform query` workflow. A Microsoft Entra ID Cloud
// Identity Provider is created as the backing resource (no env gate required):
// an Entra ID connection needs only a directory id, and nothing in that block is
// write-only, so the whole resource can be read back. The tests pin the
// name_substring filter client-side behaviour and the include_resource path,
// which since issue #379 reads each provider individually to fill in the
// settings block the discriminator requires.

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
//
// Two of step 2's assertions carry issue #379. The settings block is the point
// of it: a listing reporting only the display and provider names generates a
// resource whose own discriminator then refuses it, because
// `provider_name = "ENTRA_ID"` requires the block — so the directory id is what
// proves the per-item read happened rather than the summary being passed
// through. And mappings must stay ABSENT: Jamf Pro returns generated mappings
// for every connection, and writing them into a generated configuration would
// plan an add against the null that importing the same provider produces.
// entraDirectoryID is the Microsoft Entra directory (tenant) id the backing
// fixtures declare. Jamf Pro stores it without contacting Microsoft, so any
// well-formed GUID serves, and naming it once lets the include_resource assertions
// check the value the per-item read returned rather than restating a literal.
const entraDirectoryID = "d5749c84-5cc5-4691-a187-4545c02ff915"

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
    tenant_id = %q
  }
}
`, displayName, entraDirectoryID),
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
							{
								Path:       tfjsonpath.New("entra_id").AtMapKey("tenant_id"),
								KnownValue: knownvalue.StringExact(entraDirectoryID),
							},
							{Path: tfjsonpath.New("entra_id").AtMapKey("mappings"), KnownValue: knownvalue.Null()},
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
    tenant_id = %q
  }
}
`, displayName, entraDirectoryID),
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
