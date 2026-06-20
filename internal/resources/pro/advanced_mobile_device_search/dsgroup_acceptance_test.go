// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

package advanced_mobile_device_search_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/testhelpers"
)

// Directory-service group criteria coverage. This is the highest-value DS
// acceptance: criteria is a types.List (Optional+Computed), so the resolve /
// readback / suppress paths bridge types.List <-> []CriterionModel — exercise
// that round-trip end to end. See the computer-search counterpart for rationale.
// Stands up the shared Okta LDAP directory-service fixture (via the SDK, so the
// directory exists before the pre-apply group resolve) and resolves
// JAMFPLATFORM_ACC_LDAP_GROUP_NAME against it.

// TestAccResource_AMDS_DSGroupCriteria mirrors the computer-search test on the
// mobile (Pro v1, types.List) surface.
func TestAccResource_AMDS_DSGroupCriteria(t *testing.T) {
	ldapEnv := testhelpers.RequireOktaLdapEnv(t)
	groupName := testhelpers.RequireLdapGroupName(t)
	testhelpers.AccPreCheck(t)
	// The "Username directory service group" / "member of" criterion is rejected
	// before Jamf Pro 11.29.
	testhelpers.RequireMinJamfProVersion(t, "11.29.0")

	suffix := testhelpers.RunSuffix()
	name := "tf-acc-amds-dsgroup-" + suffix
	testhelpers.EnsureLdapServerFixture(t, name, ldapEnv)
	groupValue := testhelpers.ResolveDSGroupWireValue(t, groupName)
	const rn = "jamfplatform_pro_advanced_mobile_device_search.dsgroup"

	cfg := func(value string) string {
		return fmt.Sprintf(`
			resource "jamfplatform_pro_advanced_mobile_device_search" "dsgroup" {
				name = %q
				criteria = [{
					name        = "Username directory service group"
					search_type = "member of"
					value       = %q
				}]
			}
		`, name, value)
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckAMDSDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: cfg(groupName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(rn, "id"),
					resource.TestCheckResourceAttr(rn, "criteria.0.value", groupName),
				),
			},
			{
				Config:           cfg(groupValue),
				ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}},
			},
			{
				Config:           cfg(groupName),
				ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}},
			},
		},
	})
}
