// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

package blueprint_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/testhelpers"
)

// regexpPolicyVersionNotFound matches the platform's refusal of a blueprint that names a policy
// version it cannot serve. The same code covers an unknown policy and an unknown version, so the
// wording is deliberately about the reference rather than about which half is wrong.
var regexpPolicyVersionNotFound = regexp.MustCompile(`POLICY_VERSION_NOT_FOUND`)

// TestAccResource_Blueprint_AIGovernance is the end-to-end path an operator follows to govern an AI
// tool from Terraform: author a policy, publish it, and deliver that published version through a
// blueprint's AI Governance component.
//
// It is deliberately one test rather than two, because the two halves only mean anything together —
// a policy nothing delivers configures no device, and the component cannot be written without a
// published version to point at.
//
// The second step changes the policy's settings, which publishes version 2 and moves the blueprint's
// pinned version with it. That is the interesting case: the platform refuses a blueprint referencing
// a version that does not exist, so a configuration that advanced the policy without advancing the
// blueprint would fail — which is exactly why the version is interpolated rather than written out.
//
// The blueprint is left undeployed. Deployment with this component was verified separately by wire
// probe on 2026-08-30 (deploy accepted, deploymentState SUCCEEDED); leaving `deployed = false` here
// keeps the suite from pushing AI tool configuration onto the sandbox's real enrolled Macs.
func TestAccResource_Blueprint_AIGovernance(t *testing.T) {
	testhelpers.AccPreCheckAIGovernance(t)
	tool := testhelpers.RequireAIGovernanceTool(t, "com.anthropic.claudecode")
	suffix := testhelpers.RunSuffix()
	policyName := "tf-acc-bp-ai-policy-" + suffix
	blueprintName := "tf-acc-bp-ai-" + suffix

	config := func(settings string) string {
		return testBlueprintConfig(smartGroupHCL("aigov"), fmt.Sprintf(`
			resource "jamfplatform_ai_governance_policy" "governed" {
				name           = %q
				description    = "Acceptance test — safe to delete"
				tool_id        = %q
				schema_version = %q
				settings_json  = %s
			}

			resource "jamfplatform_blueprints_blueprint" "ai" {
				name          = %q
				description   = "Acceptance test — safe to delete"
				deployed      = false
				device_groups = [jamfplatform_device_group.scope.id]

				component_blocks = [
					{
						name = "AI Governance"
						ai_governance = {
							policies = [
								{
									policy_id = jamfplatform_ai_governance_policy.governed.id
									version   = jamfplatform_ai_governance_policy.governed.published_version
								},
							]
						}
					},
				]
			}
		`, policyName, tool.ID, tool.SchemaVersion, settings, blueprintName))
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckBlueprintResourcesDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: config(`jsonencode({ verbose = true })`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("jamfplatform_ai_governance_policy.governed", "published_version", "1"),
					resource.TestCheckResourceAttr("jamfplatform_blueprints_blueprint.ai", "component_blocks.#", "1"),
					resource.TestCheckResourceAttr("jamfplatform_blueprints_blueprint.ai", "component_blocks.0.name", "AI Governance"),
					resource.TestCheckResourceAttr("jamfplatform_blueprints_blueprint.ai", "component_blocks.0.ai_governance.policies.#", "1"),
					resource.TestCheckResourceAttrPair(
						"jamfplatform_blueprints_blueprint.ai", "component_blocks.0.ai_governance.policies.0.policy_id",
						"jamfplatform_ai_governance_policy.governed", "id"),
					resource.TestCheckResourceAttr("jamfplatform_blueprints_blueprint.ai", "component_blocks.0.ai_governance.policies.0.version", "1"),
				),
			},
			{
				Config: config(`jsonencode({ verbose = false, model = "sonnet" })`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("jamfplatform_ai_governance_policy.governed", "published_version", "2"),
					resource.TestCheckResourceAttr("jamfplatform_blueprints_blueprint.ai", "component_blocks.0.ai_governance.policies.0.version", "2"),
				),
			},
			{
				ResourceName:      "jamfplatform_blueprints_blueprint.ai",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// TestAccResource_Blueprint_AIGovernance_RejectsAnUnpublishedPolicy pins the platform's referential
// check, so the failure an operator sees names the policy reference rather than the whole blueprint.
//
// A policy with `publish = false` has no published version at all, so interpolating
// `published_version` yields a null the blueprint cannot use.
func TestAccResource_Blueprint_AIGovernance_RejectsAnUnpublishedPolicy(t *testing.T) {
	testhelpers.AccPreCheckAIGovernance(t)
	tool := testhelpers.RequireAIGovernanceTool(t, "com.anthropic.claudecode")
	suffix := testhelpers.RunSuffix()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckBlueprintResourcesDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: testBlueprintConfig(smartGroupHCL("aigovdraft"), fmt.Sprintf(`
					resource "jamfplatform_ai_governance_policy" "draft" {
						name           = "tf-acc-bp-ai-draft-%s"
						tool_id        = %q
						schema_version = %q
						settings_json  = jsonencode({ verbose = true })
						publish        = false
					}

					resource "jamfplatform_blueprints_blueprint" "ai_draft" {
						name          = "tf-acc-bp-ai-draft-%s"
						deployed      = false
						device_groups = [jamfplatform_device_group.scope.id]

						component_blocks = [
							{
								name = "AI Governance"
								ai_governance = {
									policies = [
										{
											policy_id = jamfplatform_ai_governance_policy.draft.id
											version   = 1
										},
									]
								}
							},
						]
					}
				`, suffix, tool.ID, tool.SchemaVersion, suffix)),
				ExpectError: regexpPolicyVersionNotFound,
			},
		},
	})
}
