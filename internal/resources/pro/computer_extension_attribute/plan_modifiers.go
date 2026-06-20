// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package computer_extension_attribute

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// popupChoicesPlanModifier is an input_type-aware UseStateForUnknown for
// popup_menu_choices. The field is Optional+Computed on a full-replace endpoint,
// but it is also discriminator-gated: it is only valid when input_type = POPUP.
//
// A plain UseStateForUnknown would carry the prior choices into the plan whenever
// the field is omitted — including on a POPUP→other transition, where the server
// clears the choices ([] → null). That mismatch (plan predicts the old set, apply
// yields null) trips "Provider produced inconsistent result after apply".
//
// This modifier only fires when the value is omitted (plan Unknown):
//   - input_type still POPUP → behave like UseStateForUnknown (carry the prior
//     value in so an omitted field is preserved, not cleared).
//   - input_type no longer POPUP → predict SetNull, matching the cleared transition
//     result (the input builder drops the field for non-POPUP types).
//   - input_type unknown (interpolated) → leave Unknown; resolve at apply.
type popupChoicesPlanModifier struct{}

func (popupChoicesPlanModifier) Description(context.Context) string {
	return "Preserves popup_menu_choices when omitted on a POPUP EA; clears it when input_type changes away from POPUP."
}

func (m popupChoicesPlanModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (popupChoicesPlanModifier) PlanModifySet(ctx context.Context, req planmodifier.SetRequest, resp *planmodifier.SetResponse) {
	// Only act when the user omitted the attribute (Computed → Unknown in plan).
	// A configured value is already known and must be honoured as-is.
	if !req.PlanValue.IsUnknown() {
		return
	}

	var inputType types.String
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("input_type"), &inputType)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if inputType.IsNull() || inputType.IsUnknown() {
		// Cannot decide yet; leave Unknown so apply resolves it.
		return
	}

	if inputType.ValueString() != inputTypePopup {
		// Not a POPUP EA: the choices are cleared server-side, so predict null.
		resp.PlanValue = types.SetNull(types.StringType)
		return
	}

	// POPUP EA with the field omitted: preserve the prior value (omit = preserve).
	if !req.StateValue.IsNull() && !req.StateValue.IsUnknown() {
		resp.PlanValue = req.StateValue
	}
	// Otherwise leave Unknown — an empty POPUP resolves to the server's [] (null).
}

var _ planmodifier.Set = popupChoicesPlanModifier{}
