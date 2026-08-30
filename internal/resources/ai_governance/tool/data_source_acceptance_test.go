// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

package tool_test

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/testhelpers"
)

// TestAccDataSource_AIGovernanceTools reads the catalogue and one tool from it, including the schema
// document a policy's settings are written against.
//
// The assertions are deliberately shape-based rather than value-based: which tools Jamf offers and
// which schema versions each carries is platform data that changes without notice, so pinning
// com.anthropic.claudecode's current version here would break the suite on a Jamf release.
func TestAccDataSource_AIGovernanceTools(t *testing.T) {
	testhelpers.AccPreCheckAIGovernance(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
					data "jamfplatform_ai_governance_tools" "all" {}

					data "jamfplatform_ai_governance_tool" "first" {
						id = data.jamfplatform_ai_governance_tools.all.tools[0].id
					}
				`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.jamfplatform_ai_governance_tools.all", "tools.#"),
					resource.TestCheckResourceAttrSet("data.jamfplatform_ai_governance_tools.all", "tools.0.id"),
					resource.TestCheckResourceAttrSet("data.jamfplatform_ai_governance_tools.all", "tools.0.display_name"),
					resource.TestCheckResourceAttrSet("data.jamfplatform_ai_governance_tools.all", "tools.0.current_schema_version"),
					resource.TestCheckResourceAttrSet("data.jamfplatform_ai_governance_tools.all", "tools.0.schema_versions.#"),

					resource.TestCheckResourceAttrSet("data.jamfplatform_ai_governance_tool.first", "display_name"),
					resource.TestCheckResourceAttrSet("data.jamfplatform_ai_governance_tool.first", "settings_schema_json"),
					// With no schema_version set, the data source reads the tool's current one.
					resource.TestCheckResourceAttrPair(
						"data.jamfplatform_ai_governance_tool.first", "schema_version",
						"data.jamfplatform_ai_governance_tool.first", "current_schema_version"),
					checkSchemaLooksLikeJSONSchema("data.jamfplatform_ai_governance_tool.first"),
				),
			},
		},
	})
}

// checkSchemaLooksLikeJSONSchema asserts the served document is a JSON Schema rather than, say, an
// error body rendered as a string.
func checkSchemaLooksLikeJSONSchema(name string) resource.TestCheckFunc {
	return resource.ComposeAggregateTestCheckFunc(
		resource.TestMatchResourceAttr(name, "settings_schema_json", regexp.MustCompile(`"\$schema"`)),
		resource.TestMatchResourceAttr(name, "settings_schema_json", regexp.MustCompile(`"type"\s*:\s*"object"`)),
	)
}

// TestAccDataSource_AIGovernanceTool_UnknownSchemaVersion covers the version check: a version the
// tool does not offer must name the ones it does, rather than surfacing a bare read failure.
func TestAccDataSource_AIGovernanceTool_UnknownSchemaVersion(t *testing.T) {
	testhelpers.AccPreCheckAIGovernance(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
					data "jamfplatform_ai_governance_tools" "all" {}

					data "jamfplatform_ai_governance_tool" "old" {
						id             = data.jamfplatform_ai_governance_tools.all.tools[0].id
						schema_version = "1999-01-01"
					}
				`,
				ExpectError: regexp.MustCompile(`Accepted\s+versions`),
			},
		},
	})
}

// TestAccDataSource_AIGovernanceTool_UnknownTool covers a tool identifier that is not in the
// catalogue.
func TestAccDataSource_AIGovernanceTool_UnknownTool(t *testing.T) {
	testhelpers.AccPreCheckAIGovernance(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
					data "jamfplatform_ai_governance_tool" "missing" {
						id = "com.example.nope"
					}
				`,
				ExpectError: regexp.MustCompile(`AI\s+tool\s+not\s+found`),
			},
		},
	})
}
