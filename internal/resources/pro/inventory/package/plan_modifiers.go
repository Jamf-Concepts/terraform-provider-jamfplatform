// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package pkg

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

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

// resetIfSourceChangedString is a plan modifier for Optional+Computed
// string attributes whose server-derived value depends on the bytes of a
// `*_file_source` upload input. When the watched source attribute differs
// between state and plan, the attribute's plan value is left Unknown so
// the server can populate a fresh value during apply. When the source is
// unchanged, the prior state value carries forward (UseStateForUnknown
// semantics) so trivial metadata updates don't churn the diff.
//
// Required for: sha3_512, sha256, md5, hash_type, hash_value, size,
// cloud_transfer_status (watch package_file_source); manifest,
// manifest_file_name (watch manifest_file_source). Without this modifier
// every JCDS re-upload trips Terraform's "provider produced inconsistent
// result after apply" check because UseStateForUnknown carries the old
// hash into the plan while apply legitimately writes the new one.
func resetIfSourceChangedString(sourcePaths ...path.Expression) planmodifier.String {
	return resetStringIfSourceChanged{sources: sourcePaths}
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

	if override, ok := decideResetForUnchangedString(allEqual, req.StateValue); ok {
		resp.PlanValue = override
	}
}

// decideResetForUnchangedString centralises the override decision for the
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
func decideResetForUnchangedString(sourceEqual bool, stateValue types.String) (types.String, bool) {
	if !sourceEqual {
		return types.StringNull(), false
	}
	if stateValue.IsNull() || stateValue.IsUnknown() {
		return types.StringNull(), false
	}
	return stateValue, true
}
