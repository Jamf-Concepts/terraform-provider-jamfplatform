// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package macos_configuration_profile

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// payloadsDiffSuppressor returns a string plan modifier that suppresses
// diffs on the `general.payloads` attribute when the user's plan value is
// semantically equivalent to the current state value modulo Jamf Pro's
// well-known server-side normalisations. See helpers.go for the mask logic
// and PROFILE_ROUNDTRIP_REPORT.md §3 for the full diff-class catalogue
// the suppression neutralises.
//
// When the plan and state payloads are semantically equal, the plan is
// rewritten to match state so Terraform considers the attribute unchanged.
// When the plan is genuinely different, the modifier leaves the plan value
// alone — Terraform will surface the change as drift.
//
// The TF_LOG=DEBUG path always prints both the suppression decision and
// (when suppressing) a coarse byte-length / structural summary so operators
// debugging unexpected silence can see what the mask is doing.
func payloadsDiffSuppressor() planmodifier.String {
	return payloadsDiffSuppressorModifier{}
}

type payloadsDiffSuppressorModifier struct{}

func (m payloadsDiffSuppressorModifier) Description(_ context.Context) string {
	return "Suppresses diffs on the mobileconfig payload when the user-supplied plan matches the server-canonical state modulo Jamf Pro normalisations."
}

func (m payloadsDiffSuppressorModifier) MarkdownDescription(_ context.Context) string {
	return "Suppresses diffs on the mobileconfig `payloads` attribute when the user-supplied plan matches the server-canonical state modulo Jamf Pro normalisations (UUID/Identifier/DisplayName rewrites, server-injected defaults, whitespace, etc.)."
}

func (m payloadsDiffSuppressorModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	// Skip on Create — no state to compare against. The plan value is the
	// canonical form for first apply.
	if req.StateValue.IsNull() {
		return
	}
	// Skip when the user clears the field (Required schema rejects this
	// anyway, but defend in depth).
	if req.PlanValue.IsNull() || req.PlanValue.IsUnknown() {
		return
	}
	if req.StateValue.IsUnknown() {
		return
	}

	planRaw := req.PlanValue.ValueString()
	stateRaw := req.StateValue.ValueString()
	if planRaw == stateRaw {
		tflog.Debug(ctx, "payload diff: byte-equal, no action needed")
		return
	}

	equal, err := payloadsSemanticallyEqual([]byte(planRaw), []byte(stateRaw))
	if err != nil {
		resp.Diagnostics.AddAttributeWarning(req.Path,
			"Payload diff comparison failed; falling through to byte-level diff",
			fmt.Sprintf("Could not parse or mask the payload for diff suppression: %v. Terraform will treat the change as drift.", err),
		)
		return
	}
	if !equal {
		tflog.Info(ctx, "payload diff: state and plan differ after masking; surfacing as drift",
			map[string]any{
				"plan_bytes":  len(planRaw),
				"state_bytes": len(stateRaw),
			})
		return
	}
	tflog.Info(ctx, "payload diff suppressed: state and plan semantically equal after masking",
		map[string]any{
			"plan_bytes":  len(planRaw),
			"state_bytes": len(stateRaw),
		})
	resp.PlanValue = req.StateValue
}
