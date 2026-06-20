// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package ibeacon

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// includeAnyMajorMinorConfigValidator enforces the iBeacon cross-field rules
// at plan time, before apply. The apply-time helper validateIbeaconPlan is
// retained as defence-in-depth (catches values that only become known during
// apply).
//
// Off-the-shelf framework validators (e.g. boolvalidator.AlsoRequires) do not
// support value-based discrimination: they fire when an attribute is set, not
// when it equals a specific value of another attribute. The rule here is
// "major required only when include_any_major_value is false (or omitted)"
// — value-based, and the same independently for minor — so we implement a
// custom resource.ConfigValidator. See STYLE_GUIDE §Cross-field validation.
type includeAnyMajorMinorConfigValidator struct{}

// Description returns a plain-text description of the validator.
func (includeAnyMajorMinorConfigValidator) Description(context.Context) string {
	return "include_any_major_value and include_any_minor_value are independent — each true value forbids its own axis; each false value (or null) requires its own axis"
}

// MarkdownDescription returns the markdown description.
func (v includeAnyMajorMinorConfigValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

// ValidateResource implements the plan-time cross-field check. ConfigValidators
// run on the user's config, so include_any_*_value is null when the user omits
// it (the schema Default has not been applied yet). Treat null as false.
func (includeAnyMajorMinorConfigValidator) ValidateResource(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data IbeaconResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Major axis.
	if !data.IncludeAnyMajorValue.IsUnknown() && !data.Major.IsUnknown() {
		includeAnyMajor := !data.IncludeAnyMajorValue.IsNull() && data.IncludeAnyMajorValue.ValueBool()
		majorSet := !data.Major.IsNull()
		switch {
		case includeAnyMajor && majorSet:
			resp.Diagnostics.AddAttributeError(
				path.Root("major"),
				"major forbidden when include_any_major_value = true",
				"Set either `include_any_major_value = true` (match any major) OR a concrete `major` value — not both.",
			)
		case !includeAnyMajor && !majorSet:
			resp.Diagnostics.AddAttributeError(
				path.Root("major"),
				"major required",
				"`major` is required when `include_any_major_value` is unset or false. To match any major value, set `include_any_major_value = true` and omit `major`.",
			)
		}
	}

	// Minor axis. Independent of major — Jamf supports e.g. concrete major +
	// `include_any_minor_value = true` (specific major, any minor).
	if !data.IncludeAnyMinorValue.IsUnknown() && !data.Minor.IsUnknown() {
		includeAnyMinor := !data.IncludeAnyMinorValue.IsNull() && data.IncludeAnyMinorValue.ValueBool()
		minorSet := !data.Minor.IsNull()
		switch {
		case includeAnyMinor && minorSet:
			resp.Diagnostics.AddAttributeError(
				path.Root("minor"),
				"minor forbidden when include_any_minor_value = true",
				"Set either `include_any_minor_value = true` (match any minor) OR a concrete `minor` value — not both.",
			)
		case !includeAnyMinor && !minorSet:
			resp.Diagnostics.AddAttributeError(
				path.Root("minor"),
				"minor required",
				"`minor` is required when `include_any_minor_value` is unset or false. To match any minor value, set `include_any_minor_value = true` and omit `minor`.",
			)
		}
	}
}
