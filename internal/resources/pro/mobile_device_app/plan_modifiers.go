// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package mobile_device_app

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
)

// preferencesNewlineSemanticEquality suppresses a plan diff on
// app_configuration.preferences when the only difference between the prior state
// and the planned config is newline style (CRLF vs LF) or a trailing newline.
// Jamf Pro round-trips the content but strips the trailing newline and stores
// CRLF for UI-authored/imported apps; without this, an LF-authored or heredoc
// (`<<-EOT`, trailing newline) config would permadiff against the server value.
// Real content edits still surface — only those normalisations are collapsed
// (to the prior state value, so the wire bytes stay stable).
type preferencesNewlineSemanticEquality struct{}

var _ planmodifier.String = preferencesNewlineSemanticEquality{}

func (preferencesNewlineSemanticEquality) Description(_ context.Context) string {
	return "Treats CRLF/LF and a trailing newline as equivalent so those differences do not produce a diff."
}

func (m preferencesNewlineSemanticEquality) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (preferencesNewlineSemanticEquality) PlanModifyString(_ context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	// No prior state (create) or unknown either side: leave the framework's
	// default planned value untouched.
	if req.StateValue.IsNull() || req.StateValue.IsUnknown() {
		return
	}
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	if preferencesEqual(req.StateValue.ValueString(), req.ConfigValue.ValueString()) {
		resp.PlanValue = req.StateValue
	}
}
