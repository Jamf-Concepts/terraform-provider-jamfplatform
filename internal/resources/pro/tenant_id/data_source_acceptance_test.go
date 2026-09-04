// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

package tenant_id_test

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/testhelpers"
)

// uuidPattern matches the canonical hyphenated UUID form the tenant identifier
// takes. Asserting the shape rather than a literal is the point: the identifier
// differs per tenant, so a fixed value would pin the test to one estate, while
// asserting merely "not empty" would pass on a truncated or placeholder value.
var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// TestAccDataSource_ProTenantID_Basic reads the tenant identifier for the
// configured scope.
//
// The data source takes no arguments and has no update or import path, so a
// single step is the whole surface — there is no attribute a caller can vary and
// nothing to round-trip. What is worth asserting is that the value arrives in the
// shape a consumer needs, since its only purpose is to be handed to another
// construct that will reject a malformed one.
func TestAccDataSource_ProTenantID_Basic(t *testing.T) {
	testhelpers.AccPreCheck(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `data "jamfplatform_pro_tenant_id" "test" {}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.jamfplatform_pro_tenant_id.test", "id", "singleton"),
					resource.TestMatchResourceAttr("data.jamfplatform_pro_tenant_id.test", "tenant_id", uuidPattern),
				),
			},
		},
	})
}

// TestAccDataSource_ProTenantID_MatchesConfiguredScope pins the claim the data
// source's whole value rests on: that what it returns is the tenant the provider
// is scoped to, not some other identifier Jamf happens to hold.
//
// Only meaningful under a tenant-scoped integration, where the configured value
// is the expected answer. Under an environment scope the provider sends an
// environment identifier and the tenant is resolved server-side, so there is
// nothing local to compare against and the test skips rather than asserting
// something it cannot know.
func TestAccDataSource_ProTenantID_MatchesConfiguredScope(t *testing.T) {
	testhelpers.AccPreCheck(t)

	tenant := testhelpers.AccTenantIDOrSkip(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `data "jamfplatform_pro_tenant_id" "test" {}`,
				Check: resource.TestCheckResourceAttr(
					"data.jamfplatform_pro_tenant_id.test", "tenant_id", tenant),
			},
		},
	})
}
