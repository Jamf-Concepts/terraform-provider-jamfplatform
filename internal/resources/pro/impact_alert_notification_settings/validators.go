// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package impact_alert_notification_settings

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// confirmationCodeRequiresAlertValidator enforces, at plan time, the dependency Jamf Pro
// enforces on the wire: a `*_confirmation_code_enabled` toggle requires its matching
// `*_alert_enabled` toggle to be `true`. Wire-probed 2026-06-09 — both pairs return HTTP
// 400 (`*_CONFIRMATION_CODE_ENABLED_WHEN_ALERT_DISABLED`) when a confirmation code is set
// while its alert is off.
//
// Coverage is intentionally PARTIAL: a ConfigValidator reads the *config*, not the
// resolved plan, so it fires only when BOTH fields of a pair are explicitly declared
// (confirmation_code = true, alert = false). The applied confirmation-code value can also
// arrive via UseStateForUnknown (update) or the create-adopt GET — neither visible here —
// so any violation where the offending field is omitted/preserved falls through to the
// server 400 as the backstop. The schema MarkdownDescription documents the footgun (to
// turn an alert off, set its matching confirmation-code toggle to false in the same
// apply). Each pair defers (no error) when either field is null or unknown, per
// STYLE_GUIDE §"Config-time validators MUST defer on unknown values" (and because a null
// value means "preserve the current server value", which may already satisfy the rule).
type confirmationCodeRequiresAlertValidator struct{}

// Description returns a plain-text description of the validator.
func (confirmationCodeRequiresAlertValidator) Description(context.Context) string {
	return "A *_confirmation_code_enabled toggle requires its matching *_alert_enabled toggle to be true."
}

// MarkdownDescription returns the markdown description.
func (v confirmationCodeRequiresAlertValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

// ValidateResource implements the plan-time cross-field check for both object pairs.
func (confirmationCodeRequiresAlertValidator) ValidateResource(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	checkPair(ctx, req, resp, "deployable_objects_alert_enabled", "deployable_objects_confirmation_code_enabled")
	checkPair(ctx, req, resp, "scopeable_objects_alert_enabled", "scopeable_objects_confirmation_code_enabled")
}

// checkPair errors when confirmationAttr is explicitly true while alertAttr is explicitly
// false. Defers (returns without error) when either field is null or unknown.
func checkPair(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse, alertAttr, confirmationAttr string) {
	// Collect GetAttribute diagnostics locally: resp.Diagnostics may already carry an
	// error from the other pair, so guarding on resp.Diagnostics.HasError() here would
	// skip the second pair's check after the first pair errors.
	var alert, confirmation types.Bool
	diags := req.Config.GetAttribute(ctx, path.Root(alertAttr), &alert)
	diags.Append(req.Config.GetAttribute(ctx, path.Root(confirmationAttr), &confirmation)...)
	resp.Diagnostics.Append(diags...)
	if diags.HasError() {
		return
	}

	if alert.IsNull() || alert.IsUnknown() || confirmation.IsNull() || confirmation.IsUnknown() {
		return
	}

	if confirmation.ValueBool() && !alert.ValueBool() {
		resp.Diagnostics.AddAttributeError(
			path.Root(confirmationAttr),
			"Confirmation code requires its alert",
			fmt.Sprintf("%q requires %q to be true. Jamf Pro rejects a confirmation code while its alert is disabled.", confirmationAttr, alertAttr),
		)
	}
}
