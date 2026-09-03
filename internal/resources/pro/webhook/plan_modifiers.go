// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package webhook

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// hashAlgorithmAuthResetPlanModifier handles the server's discriminator-reset
// of <hash_algorithm>: the value is only meaningful for HASH_SIGNATURE auth, and
// the server FORCES it back to the SHA256 default on any write whose
// authentication_type is not HASH_SIGNATURE (wire-probed — WEBHOOK_SPIKE.md §5).
//
// A plain UseStateForUnknown is wrong here: Terraform core proposes the prior
// state value (e.g. SHA512) for an unset Optional+Computed attribute, but the
// server resets it to SHA256 when auth changes away from HASH_SIGNATURE → a
// "produced inconsistent result after apply" error (same family as the derived
// *_name trap). This modifier:
//
//   - honours an explicitly-configured value (no override);
//   - on create (no prior state), leaves the proposal untouched;
//   - when authentication_type is UNCHANGED, reuses the prior state value so the
//     plan stays stable (no perpetual "known after apply");
//   - when authentication_type is CHANGING, marks the value unknown so the
//     server's (possibly reset) hash_algorithm is accepted post-apply.
type hashAlgorithmAuthResetPlanModifier struct{}

func (hashAlgorithmAuthResetPlanModifier) Description(context.Context) string {
	return "reuses prior state for hash_algorithm unless authentication_type changes, in which case the value Jamf Pro assigns is accepted"
}

func (m hashAlgorithmAuthResetPlanModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (hashAlgorithmAuthResetPlanModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	var planAuth, stateAuth types.String
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("authentication_type"), &planAuth)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("authentication_type"), &stateAuth)...)
	if resp.Diagnostics.HasError() {
		return
	}

	switch hashAlgorithmPlanDecision(req.ConfigValue.IsNull(), req.StateValue.IsNull(), planAuth, stateAuth) {
	case hashAlgoUnknownOnAuthChange:
		resp.PlanValue = types.StringUnknown()
	case hashAlgoReuseState:
		resp.PlanValue = req.StateValue
	default:
		// hashAlgoHonorConfig / hashAlgoLeaveCreate: leave the proposal as-is.
	}
}

// hashAlgoPlanAction enumerates the plan-time outcomes for hash_algorithm.
type hashAlgoPlanAction int

const (
	hashAlgoHonorConfig         hashAlgoPlanAction = iota // explicitly configured → honour it
	hashAlgoLeaveCreate                                   // no prior state → server assigns default
	hashAlgoUnknownOnAuthChange                           // auth changing → accept server reset
	hashAlgoReuseState                                    // auth stable → reuse prior state (no churn)
)

// hashAlgorithmPlanDecision is the pure decision behind
// hashAlgorithmAuthResetPlanModifier, factored out for unit testing.
func hashAlgorithmPlanDecision(configNull, stateNull bool, planAuth, stateAuth types.String) hashAlgoPlanAction {
	if !configNull {
		return hashAlgoHonorConfig
	}
	if stateNull {
		return hashAlgoLeaveCreate
	}
	if planAuth.IsUnknown() || !planAuth.Equal(stateAuth) {
		return hashAlgoUnknownOnAuthChange
	}
	return hashAlgoReuseState
}
