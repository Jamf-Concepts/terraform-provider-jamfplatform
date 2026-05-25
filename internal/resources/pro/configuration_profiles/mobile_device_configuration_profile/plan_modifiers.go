// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package mobile_device_configuration_profile

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/configuration_profiles/payloadhelpers"
)

// payloadsDiffSuppressor returns a string plan modifier that suppresses
// diffs on the `general.payloads` attribute when the user's plan value is
// semantically equivalent to the current state value modulo Jamf Pro's
// well-known server-side normalisations. See helpers.go for the mask logic.
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
	if req.StateValue.IsNull() {
		return
	}
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

	equal, err := payloadhelpers.PayloadsSemanticallyEqual([]byte(planRaw), []byte(stateRaw))
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
