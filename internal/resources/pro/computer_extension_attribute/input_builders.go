// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package computer_extension_attribute

import (
	"context"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/pro"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
)

// buildComputerExtensionAttributeInput converts a plan model into the SDK
// payload used for Create and Update. The Pro /v1 PUT is a full replace — every
// omitted optional field is cleared server-side.
//
// The three input_type-gated companion fields (script, directory_service_attribute,
// popup_menu_choices) are sent ONLY when their input_type is active. This keeps a
// transition clean: on e.g. SCRIPT→TEXT the script is not sent, so full-replace
// clears it. It is also what lets popup_menu_choices be Optional+Computed without
// breaking transitions — its UseStateForUnknown plan modifier may carry a prior
// value into the plan, but the gate here drops it whenever input_type is no longer
// POPUP, so a stale foreign value can never reach the wire (which would 400).
// manageExistingData is the WriteOnly value, read from config by the caller (it
// is null in plan/state, so it cannot come from `plan`). isCreate must be true
// only when building the Create payload: Jamf Pro 400s
// ("This field should be blank for first time CEA creation.") if
// manageExistingData is present on create, and only accepts it on update.
func buildComputerExtensionAttributeInput(ctx context.Context, plan ComputerExtensionAttributeResourceModel, manageExistingData types.String, isCreate bool) (*pro.ComputerExtensionAttributes, diag.Diagnostics) {
	var diags diag.Diagnostics

	inputType := plan.InputType.ValueString()

	ea := &pro.ComputerExtensionAttributes{
		Name:                          plan.Name.ValueString(),
		DataType:                      plan.DataType.ValueString(),
		InputType:                     inputType,
		InventoryDisplayType:          plan.InventoryDisplay.ValueString(),
		Description:                   helpers.OptionalStringPointer(plan.Description),
		Enabled:                       helpers.OptionalBoolPointer(plan.Enabled),
		LdapExtensionAttributeAllowed: helpers.OptionalBoolPointer(plan.AllowMultipleValues),
	}

	switch inputType {
	case inputTypeScript:
		ea.ScriptContents = helpers.OptionalStringPointer(plan.Script)
		// manageExistingData is required (and only valid) for SCRIPT EAs on
		// update: Jamf Pro 400s a SCRIPT update without it, and 400s a SCRIPT
		// create with it present. Send the user's value, or RETAIN when
		// omitted, but only on update. Never send it for non-SCRIPT EAs or on
		// create.
		if !isCreate {
			mode := manageExistingDataDefault
			if v := helpers.OptionalStringPointer(manageExistingData); v != nil {
				mode = *v
			}
			ea.ManageExistingData = &mode
		}
	case inputTypeLDAP:
		ea.LdapAttributeMapping = helpers.OptionalStringPointer(plan.DirectoryServiceAttribute)
	case inputTypePopup:
		if !plan.PopupMenuChoices.IsNull() && !plan.PopupMenuChoices.IsUnknown() {
			choices := []string{}
			diags.Append(plan.PopupMenuChoices.ElementsAs(ctx, &choices, false)...)
			if diags.HasError() {
				return nil, diags
			}
			ea.PopupMenuChoices = &choices
		}
	}

	return ea, diags
}
