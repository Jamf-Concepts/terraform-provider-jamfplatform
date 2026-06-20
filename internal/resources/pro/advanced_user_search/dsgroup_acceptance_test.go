// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

package advanced_user_search_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/testhelpers"
)

// Directory-service group criteria coverage — gated behind
// JAMFPLATFORM_ACC_CRITERIA_DS_GROUP=1 (+ _NAME = an exact LDAP/cloud-IdP group
// name). The user surface accepts only the Username directory-service group
// criterion. See the computer-search counterpart for the full rationale.
const (
	envDSGroupGate = "JAMFPLATFORM_ACC_CRITERIA_DS_GROUP"
	envDSGroupName = "JAMFPLATFORM_ACC_CRITERIA_DS_GROUP_NAME"
)

// TestAccResource_AUS_DSGroupCriteria mirrors the computer-search test on the
// user surface (Username criterion only).
func TestAccResource_AUS_DSGroupCriteria(t *testing.T) {
	if os.Getenv(envDSGroupGate) == "" {
		t.Skipf("set %s=1 (plus %s) to run directory-service group criteria acceptance", envDSGroupGate, envDSGroupName)
	}
	groupName := os.Getenv(envDSGroupName)
	if groupName == "" {
		t.Skipf("%s must be set", envDSGroupName)
	}
	testhelpers.AccPreCheck(t)
	groupValue := testhelpers.ResolveDSGroupWireValue(t, groupName)

	suffix := testhelpers.RunSuffix()
	name := "tf-acc-aus-dsgroup-" + suffix
	const rn = "jamfplatform_pro_advanced_user_search.dsgroup"

	cfg := func(value string) string {
		return fmt.Sprintf(`
			resource "jamfplatform_pro_advanced_user_search" "dsgroup" {
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
		CheckDestroy:             testAccCheckAUSDestroy(t),
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
