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
// omitted optional field is cleared server-side — so the complete plan
// representation is always sent. Companion fields that are invalid for the
// chosen input_type are simply left nil (the plan-time validator has already
// rejected mis-set ones), and the server clears any orphaned field on a
// transition.
// manageExistingData is the WriteOnly value, read from config by the caller (it
// is null in plan/state, so it cannot come from `plan`).
func buildComputerExtensionAttributeInput(ctx context.Context, plan ComputerExtensionAttributeResourceModel, manageExistingData types.String) (*pro.ComputerExtensionAttributes, diag.Diagnostics) {
	var diags diag.Diagnostics

	ea := &pro.ComputerExtensionAttributes{
		Name:                          plan.Name.ValueString(),
		DataType:                      plan.DataType.ValueString(),
		InputType:                     plan.InputType.ValueString(),
		InventoryDisplayType:          plan.InventoryDisplay.ValueString(),
		Description:                   helpers.OptionalStringPointer(plan.Description),
		Enabled:                       helpers.OptionalBoolPointer(plan.Enabled),
		ScriptContents:                helpers.OptionalStringPointer(plan.Script),
		LdapAttributeMapping:          helpers.OptionalStringPointer(plan.DirectoryServiceAttribute),
		LdapExtensionAttributeAllowed: helpers.OptionalBoolPointer(plan.AllowMultipleValues),
	}

	// manageExistingData is required (and only valid) for SCRIPT EAs: Jamf Pro
	// 400s a SCRIPT update without it. Send the user's value, or RETAIN when
	// omitted. Never send it for non-SCRIPT EAs.
	if plan.InputType.ValueString() == inputTypeScript {
		mode := manageExistingDataDefault
		if v := helpers.OptionalStringPointer(manageExistingData); v != nil {
			mode = *v
		}
		ea.ManageExistingData = &mode
	}

	if !plan.PopupMenuChoices.IsNull() && !plan.PopupMenuChoices.IsUnknown() {
		choices := []string{}
		diags.Append(plan.PopupMenuChoices.ElementsAs(ctx, &choices, false)...)
		if diags.HasError() {
			return nil, diags
		}
		ea.PopupMenuChoices = &choices
	}

	return ea, diags
}
