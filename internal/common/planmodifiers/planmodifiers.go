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
	return "Carry prior state forward when the watched source attribute is unchanged; otherwise leave it Unknown for the service to populate."
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

// MirrorOfString returns a plan modifier for a Computed-only string attribute
// that Jamf Pro derives from one or more SIBLING attributes the same resource
// manages — a mirror, not an independent server-assigned value.
//
// Neither of the obvious answers is right for a mirror. Plain
// UseStateForUnknown promises the value cannot change unless the practitioner
// changes this attribute, but it changes when the *source* changes, so the plan
// carries a stale value into an apply that returns a fresh one and Terraform
// reports "Provider produced inconsistent result after apply". Omitting the
// modifier entirely leaves the attribute Unknown on every plan, which makes
// `terraform plan -refresh=false` permanently dirty for a value nobody touched.
//
// So: carry the prior value forward while every watched source is unchanged
// between state and plan, and go Unknown as soon as one of them moves. The
// sources may be of any type — pass StringSource or BoolSource per path.
//
// Used by jamfplatform_pro_mobile_device_app for general.description (mirrors
// self_service.self_service_description) and general.deployment_type (mirrors
// general.deploy_automatically).
func MirrorOfString(sources ...SourceComparer) planmodifier.String {
	return mirrorOfString{sources: sources}
}

// SourceComparer reports whether one watched source attribute is unchanged by
// this plan. ok is false when the value could not be read at all, which the
// caller treats as "cannot tell" and resolves conservatively.
type SourceComparer func(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) (unchanged, ok bool)

// StringSource watches a string attribute.
func StringSource(p path.Expression) SourceComparer { return compareSource[types.String](p) }

// BoolSource watches a bool attribute.
func BoolSource(p path.Expression) SourceComparer { return compareSource[types.Bool](p) }

// ObjectSource watches a whole single-nested block.
//
// Use it in preference to a path *into* an Optional block: when such a block is
// null, PathMatches resolves the expression only as far as the block itself, so
// a nested-attribute watcher is handed the object's path and fails with "Cannot
// use attr.Value basetypes.StringValue, only basetypes.ObjectValue is
// supported". Watching the block is coarser — any change inside it marks the
// mirror Unknown — but a wider Unknown is only ever a slightly noisier plan,
// whereas a carried-forward stale value is a broken apply.
func ObjectSource(p path.Expression) SourceComparer { return compareSource[types.Object](p) }

// compareSource builds a SourceComparer for any framework value type. The type
// parameter is what lets GetAttribute reflect into a concrete target: an
// attr.Value interface target is not something the framework can populate.
//
// The source is read from the CONFIG, not the plan. Reading a sibling from
// req.Plan looks natural and is unreliable: the plan is the proposed new state,
// where a Computed sibling with no configured value is still Unknown until its
// own plan modifier runs, and modifier order within a nested object is not
// something a caller controls. So a plan read intermittently sees Unknown,
// concludes "changed", and leaves the mirror Unknown on a plan where nothing
// moved — which is exactly the dirty-plan symptom this modifier exists to fix.
//
// The config is stable, and it answers the question that actually matters. A
// source the practitioner has not configured cannot be changing: its own
// Optional+Computed handling carries the prior value forward. A source they
// have configured is unchanged precisely when the configured value equals what
// state holds.
func compareSource[T attr.Value](p path.Expression) SourceComparer {
	return func(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) (bool, bool) {
		configPaths, diags := req.Config.PathMatches(ctx, p)
		resp.Diagnostics.Append(diags...)
		if diags.HasError() || len(configPaths) == 0 {
			return false, false
		}
		statePaths, diags := req.State.PathMatches(ctx, p)
		resp.Diagnostics.Append(diags...)
		if diags.HasError() || len(statePaths) == 0 {
			return false, false
		}
		var configSrc, stateSrc T
		if diags := req.Config.GetAttribute(ctx, configPaths[0], &configSrc); diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return false, false
		}
		if diags := req.State.GetAttribute(ctx, statePaths[0], &stateSrc); diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return false, false
		}
		// Not configured: the practitioner is not moving it.
		if configSrc.IsNull() || configSrc.IsUnknown() {
			return true, true
		}
		return configSrc.Equal(stateSrc), true
	}
}

type mirrorOfString struct{ sources []SourceComparer }

func (m mirrorOfString) Description(_ context.Context) string {
	return "Carry the prior value forward while every attribute this one mirrors is unchanged; go Unknown as soon as one of them moves."
}

func (m mirrorOfString) MarkdownDescription(ctx context.Context) string { return m.Description(ctx) }

func (m mirrorOfString) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	// Create (no prior state) and destroy (no plan) have nothing to carry.
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}

	for _, src := range m.sources {
		equal, ok := src(ctx, req, resp)
		if !ok {
			// The source could not be read. Leave the attribute Unknown: an
			// apply then writes whatever the server says, which is always
			// correct, where guessing "unchanged" risks the inconsistent-result
			// error this modifier exists to avoid.
			return
		}
		if !equal {
			return
		}
	}

	// Every source is unchanged, so the mirror cannot move: carry the prior
	// value forward. Unlike DecideResetForUnchangedString this carries a NULL
	// state value too. A mirror whose source has no value has no value either,
	// and leaving it Unknown would leave `plan -refresh=false` permanently
	// dirty on exactly the configs that never touch the source — which is most
	// of them. Unknown prior state is the one thing not carried: there is no
	// value there to carry.
	if !req.StateValue.IsUnknown() {
		resp.PlanValue = req.StateValue
	}
}
