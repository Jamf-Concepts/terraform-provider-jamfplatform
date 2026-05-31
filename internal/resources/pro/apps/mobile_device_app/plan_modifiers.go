// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package mobile_device_app

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
)

// preferencesNewlineSemanticEquality suppresses a plan diff on
// app_configuration.preferences when the only difference between the prior state
// and the planned config is newline style (CRLF vs LF). Jamf Pro round-trips
// provider-authored content verbatim, but UI-authored or imported apps store
// CRLF; without this, an LF-authored config would permadiff against a CRLF
// server value. Real content edits still surface — only newline-only differences
// are collapsed (to the prior state value, so the wire bytes stay stable).
type preferencesNewlineSemanticEquality struct{}

var _ planmodifier.String = preferencesNewlineSemanticEquality{}

func (preferencesNewlineSemanticEquality) Description(_ context.Context) string {
	return "Treats CRLF and LF newlines as equivalent so newline-only differences do not produce a diff."
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
	if normalizeNewlines(req.StateValue.ValueString()) == normalizeNewlines(req.ConfigValue.ValueString()) {
		resp.PlanValue = req.StateValue
	}
}
