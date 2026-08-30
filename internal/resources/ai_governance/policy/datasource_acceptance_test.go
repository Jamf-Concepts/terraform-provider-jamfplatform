// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

package policy_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/testhelpers"
)

// TestAccDataSource_AIGovernancePolicy reads a policy the same configuration created, so the values
// asserted are known rather than whatever the tenant happens to hold.
func TestAccDataSource_AIGovernancePolicy(t *testing.T) {
	testhelpers.AccPreCheckAIGovernance(t)
	tool := requireTool(t, "com.anthropic.claudecode")
	name := "tf-acc-ai-ds-" + testhelpers.RunSuffix()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPolicyDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "jamfplatform_ai_governance_policy" "test" {
						name           = %q
						description    = "read back by the acceptance suite"
						tool_id        = %q
						schema_version = %q
						settings_json  = jsonencode({ verbose = true })
					}

					data "jamfplatform_ai_governance_policy" "test" {
						id = jamfplatform_ai_governance_policy.test.id
					}

					data "jamfplatform_ai_governance_policies" "all" {
						sort       = ["name:asc"]
						depends_on = [jamfplatform_ai_governance_policy.test]
					}
				`, name, tool.ID, tool.SchemaVersion),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.jamfplatform_ai_governance_policy.test", "name", name),
					resource.TestCheckResourceAttr("data.jamfplatform_ai_governance_policy.test", "description", "read back by the acceptance suite"),
					resource.TestCheckResourceAttr("data.jamfplatform_ai_governance_policy.test", "tool_id", tool.ID),
					resource.TestCheckResourceAttr("data.jamfplatform_ai_governance_policy.test", "schema_version", tool.SchemaVersion),
					resource.TestCheckResourceAttr("data.jamfplatform_ai_governance_policy.test", "published_version", "1"),
					resource.TestCheckResourceAttr("data.jamfplatform_ai_governance_policy.test", "has_draft", "false"),
					resource.TestCheckResourceAttrPair(
						"data.jamfplatform_ai_governance_policy.test", "settings_json",
						"jamfplatform_ai_governance_policy.test", "settings_json"),
					resource.TestCheckResourceAttrSet("data.jamfplatform_ai_governance_policies.all", "policies.#"),
				),
			},
		},
	})
}

// TestAccDataSource_AIGovernancePolicy_NotFound covers the read of a policy that does not exist. An
// archived policy answers the same way, so this is also the archived case.
func TestAccDataSource_AIGovernancePolicy_NotFound(t *testing.T) {
	testhelpers.AccPreCheckAIGovernance(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
					data "jamfplatform_ai_governance_policy" "missing" {
						id = "00000000-0000-0000-0000-000000000000"
					}
				`,
				ExpectError: regexp.MustCompile(`AI\s+policy\s+not\s+found`),
			},
		},
	})
}

// TestAccDataSource_AIGovernancePolicies_RejectsAnUnsupportedSort pins that the sort vocabulary is
// checked at plan time. The platform refuses an unsupported property mid-read, which is a worse
// place to find out.
func TestAccDataSource_AIGovernancePolicies_RejectsAnUnsupportedSort(t *testing.T) {
	testhelpers.AccPreCheckAIGovernance(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
					data "jamfplatform_ai_governance_policies" "bad_sort" {
						sort = ["bogus:asc"]
					}
				`,
				ExpectError: regexp.MustCompile(`name,\s+createdAt,\s+updatedAt`),
			},
		},
	})
}
