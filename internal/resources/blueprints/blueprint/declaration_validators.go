// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package blueprint

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/jamf/terraform-provider-jamfplatform/internal/common/appledeclarations"
	"github.com/jamf/terraform-provider-jamfplatform/internal/common/helpers"
	"github.com/jamf/terraform-provider-jamfplatform/internal/resources/blueprints/blueprint/components"
)

// declarationSchemaValidator checks each declaration's payload against Apple's published schemas
// during plan, which is the only place these mistakes are caught.
//
// Jamf validates none of it: the blueprints service stores whatever it is given, answers 201, and a
// deploy reports SUCCEEDED — while the editor renders the offending key blank and the device never
// receives it. Every finding is therefore an error, and there is no switch to soften one. A
// declaration that should not be checked belongs in raw_component, which exists for exactly that
// and round-trips a JSON-encoded configuration unchanged.
//
// Applied to both declaration-bearing components. apple_declarations is the typed surface;
// custom_declarations delivers identical declarations with identical exposure, so leaving it
// unchecked would only relocate the silent failure.
type declarationSchemaValidator struct {
	// ordered is true for the component whose declarations are a list, where a finding can be
	// addressed to the exact element. A set has no stable index, so those findings land on the
	// collection and name the declaration type instead.
	ordered bool
}

// appleDeclarationsSchemaValidator validates the apple_declarations component.
func appleDeclarationsSchemaValidator() validator.Object {
	return declarationSchemaValidator{ordered: true}
}

// customDeclarationsSchemaValidator validates the custom_declarations component.
func customDeclarationsSchemaValidator() validator.Object {
	return declarationSchemaValidator{ordered: false}
}

func (v declarationSchemaValidator) Description(ctx context.Context) string {
	return v.MarkdownDescription(ctx)
}

func (v declarationSchemaValidator) MarkdownDescription(context.Context) string {
	_, release := appledeclarations.Provenance()
	return fmt.Sprintf(
		"each declaration's payload must match Apple's declared keys for its declaration type (schemas from %s)",
		release,
	)
}

// ValidateObject checks every declaration the component carries.
func (v declarationSchemaValidator) ValidateObject(ctx context.Context, req validator.ObjectRequest, resp *validator.ObjectResponse) {
	if !helpers.IsConfiguredValue(req.ConfigValue) {
		return
	}

	base := req.Path.AtName("declaration")

	if v.ordered {
		var component components.AppleDeclarationsComponent
		if diags := req.ConfigValue.As(ctx, &component, basetypes.ObjectAsOptions{
			UnhandledNullAsEmpty:    true,
			UnhandledUnknownAsEmpty: true,
		}); diags.HasError() {
			// A declaration still unknown at plan time cannot be read, let alone checked. The next
			// plan, once the value resolves, will check it.
			return
		}
		for i, declaration := range component.Declarations {
			resp.Diagnostics.Append(validateDeclarationPayload(
				declaration.Type, declaration.Payload,
				base.AtListIndex(i).AtName("payload"), "",
			)...)
		}
		return
	}

	var component components.CustomDeclarationsComponent
	if diags := req.ConfigValue.As(ctx, &component, basetypes.ObjectAsOptions{
		UnhandledNullAsEmpty:    true,
		UnhandledUnknownAsEmpty: true,
	}); diags.HasError() {
		return
	}
	for _, declaration := range component.Declarations {
		resp.Diagnostics.Append(validateDeclarationPayload(
			declaration.Type, declaration.Payload, base, declaration.Type.ValueString(),
		)...)
	}
}

// validateDeclarationPayload checks one declaration's payload. attribution names the declaration in
// the diagnostic when the path cannot point at it precisely, and is empty when it can.
func validateDeclarationPayload(declarationType, payload types.String, at path.Path, attribution string) diag.Diagnostics {
	var diags diag.Diagnostics

	if !helpers.IsConfiguredValue(declarationType) || !helpers.IsConfiguredValue(payload) {
		return diags
	}

	var decoded map[string]any
	if err := json.Unmarshal([]byte(payload.ValueString()), &decoded); err != nil {
		diags.AddAttributeError(at,
			"Declaration payload is not a JSON object",
			fmt.Sprintf("%sCould not decode the payload: %s. Author it with jsonencode({ ... }).",
				attributionPrefix(attribution), err),
		)
		return diags
	}

	// The kind is derived from the type rather than read from configuration, so it can never
	// disagree with the type it accompanies, and a type whose prefix Apple does not define is
	// reported once as an unknown type rather than twice.
	declared := declarationType.ValueString()
	for _, problem := range appledeclarations.Validate(appledeclarations.KindForType(declared), declared, decoded) {
		summary, detail := renderDeclarationProblem(problem, attribution)
		diags.AddAttributeError(at, summary, detail)
	}

	return diags
}

// renderDeclarationProblem turns a finding into a diagnostic, naming the schema snapshot whenever
// the finding could be caused by the provider's tables being older than what Jamf offers.
func renderDeclarationProblem(problem appledeclarations.Problem, attribution string) (summary, detail string) {
	summary = "Declaration does not match Apple's schema"
	switch problem.Kind {
	case appledeclarations.UnknownDeclarationType, appledeclarations.MiscasedDeclarationType:
		summary = "Unknown Apple declaration type"
	case appledeclarations.UnknownKey, appledeclarations.MiscasedKey:
		summary = "Unknown key in declaration payload"
	case appledeclarations.MissingRequiredKey:
		summary = "Declaration payload is missing a required key"
	case appledeclarations.UnknownStatusItem:
		summary = "Unknown status item"
	}

	parts := make([]string, 0, 5)
	if attribution != "" {
		parts = append(parts, "In declaration "+attribution+":")
	}
	if problem.Path != "" {
		parts = append(parts, "At "+problem.Path+":")
	}
	parts = append(parts, problem.Detail)

	if problem.SeedOnly {
		parts = append(parts, "This part of Apple's schema comes from a pre-release branch, so it is newer than the released one.")
	}
	if staleTableSuspect(problem.Kind) {
		parts = append(parts, fmt.Sprintf(
			"The provider's schemas come from apple/device-management %s. If Apple has published this since, "+
				"upgrade the provider; to deliver a declaration without these checks, move it to raw_component.",
			appledeclarations.ProvenanceSummary(),
		))
	}

	return summary, strings.Join(parts, " ")
}

// staleTableSuspect reports whether a finding could be caused by the embedded schemas being older
// than what Jamf offers, rather than by a genuine mistake. Only the name-based findings can be: a
// wrong value type or an out-of-range number is wrong against any version of the schema.
func staleTableSuspect(kind appledeclarations.ProblemKind) bool {
	switch kind {
	case appledeclarations.UnknownDeclarationType, appledeclarations.UnknownKey, appledeclarations.UnknownStatusItem:
		return true
	default:
		return false
	}
}

// attributionPrefix renders the optional declaration attribution as a sentence opener.
func attributionPrefix(attribution string) string {
	if attribution == "" {
		return ""
	}
	return "In declaration " + attribution + ": "
}
