// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

// Package planmodifiers provides shared Terraform Plugin Framework plan
// modifiers for use across all resource packages.
package planmodifiers

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ResetIfSourceChangedString returns a plan modifier for Optional+Computed
// string attributes whose server-derived value depends on the bytes of a
// *_file_source upload input. When any watched source attribute differs
// between state and plan, the attribute's plan value is left Unknown so
// the server can populate a fresh value during apply. When all sources are
// unchanged, the prior state value carries forward (UseStateForUnknown
// semantics) so trivial metadata updates do not churn the diff.
func ResetIfSourceChangedString(sourcePaths ...path.Expression) planmodifier.String {
	return resetStringIfSourceChanged{sources: sourcePaths}
}

// readSourceStrings extracts the watched source attribute from both plan
// and state for the string plan-modifier variant. Returns ok=false if
// either read produced a diagnostic; the modifier should bail in that case.
func readSourceStrings(ctx context.Context, req planmodifier.StringRequest, sourcePath path.Expression, resp *planmodifier.StringResponse) (planSrc, stateSrc types.String, ok bool) {
	planPaths, diags := req.Plan.PathMatches(ctx, sourcePath)
	resp.Diagnostics.Append(diags...)
	if diags.HasError() || len(planPaths) == 0 {
		return planSrc, stateSrc, false
	}
	statePaths, diags := req.State.PathMatches(ctx, sourcePath)
	resp.Diagnostics.Append(diags...)
	if diags.HasError() || len(statePaths) == 0 {
		return planSrc, stateSrc, false
	}
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, planPaths[0], &planSrc)...)
	if resp.Diagnostics.HasError() {
		return planSrc, stateSrc, false
	}
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, statePaths[0], &stateSrc)...)
	if resp.Diagnostics.HasError() {
		return planSrc, stateSrc, false
	}
	return planSrc, stateSrc, true
}

type resetStringIfSourceChanged struct{ sources []path.Expression }

func (m resetStringIfSourceChanged) Description(_ context.Context) string {
	return "Carry prior state forward when the watched source attribute is unchanged; otherwise leave Unknown so the server can populate."
}

func (m resetStringIfSourceChanged) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m resetStringIfSourceChanged) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}
	if !req.ConfigValue.IsNull() && !req.ConfigValue.IsUnknown() {
		return
	}

	allEqual := true
	for _, src := range m.sources {
		planSrc, stateSrc, ok := readSourceStrings(ctx, req, src, resp)
		if !ok {
			return
		}
		if !planSrc.Equal(stateSrc) {
			allEqual = false
			break
		}
	}

	if override, ok := DecideResetForUnchangedString(allEqual, req.StateValue); ok {
		resp.PlanValue = override
	}
}

// DecideResetForUnchangedString centralises the override decision for the
// String plan modifier so it can be unit-tested without constructing a
// full tfsdk.Plan / State. The surrounding plumbing in PlanModifyString
// (Raw null guards, config-value precedence, path reads) is covered by
// the acceptance suite.
//
// Returns (value, true) when the caller should overwrite resp.PlanValue,
// or (zero, false) to leave it untouched (default Unknown).
//
//   - sourceEqual=false → (_, false): leave Unknown; apply writes the fresh server value.
//   - sourceEqual=true + state known + state non-null → (state, true): carry forward.
//   - state null or unknown → (_, false).
func DecideResetForUnchangedString(sourceEqual bool, stateValue types.String) (types.String, bool) {
	if !sourceEqual {
		return types.StringNull(), false
	}
	if stateValue.IsNull() || stateValue.IsUnknown() {
		return types.StringNull(), false
	}
	return stateValue, true
}

// CanonicalEmptySet returns a plan modifier for Optional+Computed
// Set<String> attributes that makes an empty set the canonical "no members"
// value: a null config (attribute omitted) plans as an empty set `[]`, the same
// value an explicit `[]` config and the read path both produce. This lets
// practitioners write `attr = []` as the natural "remove everything" gesture —
// and equally omit the attribute — without tripping an "inconsistent result
// after apply" error.
//
// Why null config must plan as `[]` rather than the reverse: Terraform requires
// the planned value of an Optional+Computed attribute to equal the config value
// whenever the config is known and non-null, and `[]` is non-null — so a `[]`
// config must plan (and apply) as `[]`. The read path therefore canonicalises
// an empty wire result to `[]` (not null), and this modifier brings the null
// config into line so omission still clears (planning `[]` → the build path
// omits the empty wrapper → the server clears the category) and stays stable
// across plans (no perpetual "known after apply"). The attribute must be
// Computed for the modifier to set a value when the config is null.
//
// A non-empty config flows through unchanged; an unknown config (interpolated
// from another resource) is left to resolve at apply.
func CanonicalEmptySet() planmodifier.Set { return canonicalEmptySet{} }

type canonicalEmptySet struct{}

func (m canonicalEmptySet) Description(_ context.Context) string {
	return "Omitting the attribute clears all members, equivalent to setting []."
}

func (m canonicalEmptySet) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m canonicalEmptySet) PlanModifySet(_ context.Context, req planmodifier.SetRequest, resp *planmodifier.SetResponse) {
	// Unknown config (e.g. interpolated from another resource): leave the
	// planned value alone so it resolves during apply.
	if req.ConfigValue.IsUnknown() {
		return
	}
	// Null config (attribute omitted) settles to the canonical empty set so it
	// matches the read path and clears the category. A known config — including
	// an explicit empty `[]` — flows through unchanged (Terraform requires the
	// plan to equal a known, non-null config).
	if req.ConfigValue.IsNull() {
		resp.PlanValue = types.SetValueMust(types.StringType, []attr.Value{})
	}
}
