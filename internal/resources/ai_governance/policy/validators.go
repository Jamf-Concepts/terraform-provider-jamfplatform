// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package policy

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/aischemas"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
)

// ModifyPlan checks the tool, the schema version and the settings body against what Jamf publishes,
// during plan rather than at apply.
//
// It lives here rather than in an attribute validator because it needs the API: the tool catalogue
// and the vendor schema are server data, so there is no literal set to validate against and no
// generated SDK constant to alias. Attribute validators run before the provider is configured and
// have no client, which is what rules them out.
//
// Everything here is best-effort by construction. A catalogue or schema that cannot be fetched
// produces no findings at all — the platform validates the write regardless, and a plan that failed
// because the provider could not reach an advisory endpoint would be worse than a plan that let the
// apply report the problem.
func (r *PolicyResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() || r.client == nil {
		return
	}

	var plan policyModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !helpers.IsConfiguredValue(plan.ToolID) || !helpers.IsConfiguredValue(plan.SchemaVersion) {
		return
	}

	toolID := plan.ToolID.ValueString()
	schemaVersion := plan.SchemaVersion.ValueString()
	if !r.checkCatalogue(ctx, &resp.Diagnostics, toolID, schemaVersion) {
		return
	}
	r.checkSettings(ctx, &resp.Diagnostics, &plan, toolID, schemaVersion)
}

// checkCatalogue validates the tool and its schema version against the catalogue, and reports
// whether the pair is good enough to go on and check the settings against.
//
// A schema version behind the tool's current one is a warning rather than an error: the platform
// keeps serving it, and its own schemaDrift flag exists to say so. Announcing it at plan time is
// the point — schema_drift as a computed attribute only tells an operator after the fact.
func (r *PolicyResource) checkCatalogue(ctx context.Context, diags *diag.Diagnostics, toolID, schemaVersion string) bool {
	tool, found, err := r.schemas.Tool(ctx, toolID)
	if err != nil {
		return true
	}
	if !found {
		diags.AddAttributeError(
			path.Root("tool_id"),
			"Unknown AI tool",
			fmt.Sprintf("Jamf does not offer an AI tool with the identifier %q. %s", toolID, r.knownTools(ctx)),
		)
		return false
	}
	if !slices.Contains(tool.SchemaVersions, schemaVersion) {
		diags.AddAttributeError(
			path.Root("schema_version"),
			"Unknown settings schema version",
			fmt.Sprintf("%s does not offer a settings schema version %q. Accepted versions: %s.",
				tool.DisplayName, schemaVersion, strings.Join(tool.SchemaVersions, ", ")),
		)
		return false
	}
	if schemaVersion != tool.SchemaVersion {
		diags.AddAttributeWarning(
			path.Root("schema_version"),
			"Settings schema version is behind the current one",
			fmt.Sprintf("%s now publishes schema version %q, and this policy is written against %q. The policy "+
				"keeps working and Jamf keeps serving the older schema, but settings added since are unavailable "+
				"to it. Moving forward means setting schema_version to %q and reconciling settings_json with it.",
				tool.DisplayName, tool.SchemaVersion, schemaVersion, tool.SchemaVersion),
		)
	}
	return true
}

// knownTools renders the catalogue for a diagnostic, or says nothing when it cannot be read.
func (r *PolicyResource) knownTools(ctx context.Context) string {
	tools, err := r.schemas.Tools(ctx)
	if err != nil || len(tools) == 0 {
		return "Read the available identifiers from the jamfplatform_ai_governance_tools data source."
	}
	rendered := make([]string, 0, len(tools))
	for _, tool := range tools {
		rendered = append(rendered, fmt.Sprintf("%s (%s)", tool.ID, tool.DisplayName))
	}
	return "Available: " + strings.Join(rendered, ", ") + "."
}

// checkSettings validates the settings body against the tool's published schema.
//
// The error and warning split is the vendor schema's, not a choice: a setting of the wrong type or
// outside its accepted values is a write Jamf refuses, while an undeclared key is accepted and
// stored by a tool whose schema allows extras — and then never applied. See
// aischemas.Problem.Advisory.
func (r *PolicyResource) checkSettings(ctx context.Context, diags *diag.Diagnostics, plan *policyModel, toolID, schemaVersion string) {
	if !helpers.IsConfiguredValue(plan.SettingsJSON) {
		return
	}
	decoded, err := decodeJSON(plan.SettingsJSON.ValueString())
	if err != nil {
		return
	}
	settings, ok := decoded.(map[string]any)
	if !ok {
		return
	}

	document, err := r.schemas.Document(ctx, toolID, schemaVersion)
	if err != nil {
		return
	}

	for _, problem := range document.Validate(settings) {
		summary, detail := renderProblem(problem, schemaVersion)
		if problem.Advisory() {
			diags.AddAttributeWarning(path.Root("settings_json"), summary, detail)
			continue
		}
		diags.AddAttributeError(path.Root("settings_json"), summary, detail)
	}
}

// renderProblem turns a schema finding into a diagnostic summary and detail.
//
// The path is quoted into the detail rather than turned into a Terraform attribute path: the
// settings are one string attribute, so there is no traversable path to point at, and the JSON
// pointer is what the platform's own failures use.
func renderProblem(problem aischemas.Problem, schemaVersion string) (string, string) {
	location := "The settings"
	if problem.Path != "" {
		location = "The setting at " + problem.Path
	}

	switch problem.Kind {
	case aischemas.UnrecognisedKey:
		return "Setting is not declared for this schema version",
			problem.Detail + " Checked against schema version " + schemaVersion +
				". If this tool has published a newer schema version that declares it, set schema_version to that instead."
	case aischemas.MissingRequiredKey:
		return "Required setting is missing", location + ": " + problem.Detail
	default:
		return "Setting does not match the tool's schema",
			location + ": " + problem.Detail + " Checked against schema version " + schemaVersion + "."
	}
}
