// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

package advanced_computer_search_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/testhelpers"
)

// Directory-service group criteria coverage. The equivalent base64 is resolved +
// encoded in-test (same path the provider uses), so the swap value always matches;
// the live apply is the independent check that the encoding is server-acceptable.
// Real group names are never committed — see memory: no real LDAP names in public
// files. Stands up the shared Okta LDAP directory-service fixture (via the SDK, so
// the directory exists before the pre-apply group resolve) and resolves
// JAMFPLATFORM_ACC_PRO_LDAP_GROUP_NAME against it.

// TestAccResource_ACS_DSGroupCriteria authors a directory-service group criterion
// by NAME, asserts state round-trips back to the NAME, and asserts swapping the
// config to the equivalent raw base64 (and back) produces EMPTY plans — the
// ModifyPlan semantic-equality suppression on the []CriterionModel path.
func TestAccResource_ACS_DSGroupCriteria(t *testing.T) {
	ldapEnv := testhelpers.RequireOktaLdapEnv(t)
	groupName := testhelpers.RequireLdapGroupName(t)
	testhelpers.AccPreCheck(t)
	// The "Username directory service group" / "member of" criterion is rejected
	// before Jamf Pro 11.29.
	testhelpers.RequireMinJamfProVersion(t, "11.29.0")

	suffix := testhelpers.RunSuffix()
	name := "tf-acc-acs-dsgroup-" + suffix
	testhelpers.EnsureLdapServerFixture(t, name, ldapEnv)
	groupValue := testhelpers.ResolveDSGroupWireValue(t, groupName)
	const rn = "jamfplatform_pro_advanced_computer_search.dsgroup"

	cfg := func(value string) string {
		return fmt.Sprintf(`
			resource "jamfplatform_pro_advanced_computer_search" "dsgroup" {
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
		CheckDestroy:             testAccCheckACSDestroy(t),
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
