// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package restricted_software

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// deleteApplicationRequiresExactMatchValidator refuses
// general.delete_application on a record that does not match the exact process
// name.
//
// Jamf Pro keeps general.delete_application (wire <delete_executable>) only
// while general.restrict_exact_process_name (wire <match_exact_process_name>)
// is true. Turning the exact match off silently clears it, and a later write
// of true is accepted with HTTP 201 and discarded — wire-probed against
// 11.31.1 on 2026-09-07. The gate makes sense from the product's side: without
// an exact name Jamf Pro cannot tell which application to delete.
//
// Without this check the failure lands mid-apply as "Provider produced
// inconsistent result after apply". Both attributes are Optional+Computed, so
// it can land on a config that only turned the exact match off: a
// delete_application set by an earlier apply carries into the plan via
// UseNonNullStateForUnknown.
type deleteApplicationRequiresExactMatchValidator struct{}

// Description returns a plain-text description of the validator.
func (deleteApplicationRequiresExactMatchValidator) Description(context.Context) string {
	return "general.delete_application requires general.restrict_exact_process_name = true"
}

// MarkdownDescription returns the markdown description.
func (v deleteApplicationRequiresExactMatchValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

// ValidateResource implements the plan-time cross-field check.
//
// It reads one attribute at a time rather than the whole model: a configuration
// carrying an unknown nested value (an interpolated scope id) makes Config.Get
// on the model fail outright, which would disable every validator on the
// resource. A null or unknown restrict_exact_process_name is no violation
// either, since null takes the server default of true and unknown resolves
// after apply.
func (deleteApplicationRequiresExactMatchValidator) ValidateResource(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var deleteApplication types.Bool
	if diags := req.Config.GetAttribute(ctx, path.Root("general").AtName("delete_application"), &deleteApplication); diags.HasError() {
		return
	}
	if deleteApplication.IsNull() || deleteApplication.IsUnknown() || !deleteApplication.ValueBool() {
		return
	}

	var exactMatch types.Bool
	if diags := req.Config.GetAttribute(ctx, path.Root("general").AtName("restrict_exact_process_name"), &exactMatch); diags.HasError() {
		return
	}
	if exactMatch.IsNull() || exactMatch.IsUnknown() || exactMatch.ValueBool() {
		return
	}

	resp.Diagnostics.AddAttributeError(
		path.Root("general").AtName("delete_application"),
		"general.delete_application requires general.restrict_exact_process_name = true",
		"Jamf Pro deletes the application running a restricted process only when it can identify that application by an exact process name. "+
			"With general.restrict_exact_process_name = false it accepts the change and then clears general.delete_application without reporting an error, "+
			"which Terraform reports as \"Provider produced inconsistent result after apply\".\n\n"+
			"Set general.restrict_exact_process_name = true, or set general.delete_application = false.\n\n"+
			"general.delete_application is Optional+Computed, so a value an earlier apply set while the exact match was on carries into this plan "+
			"even when the configuration no longer mentions it. You may need to set false here to clear it.",
	)
}
