// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package mobile_device_extension_attribute

import (
	"context"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/pro"
	"github.com/hashicorp/terraform-plugin-framework/diag"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
)

// buildMobileDeviceExtensionAttributeInput converts a plan model into the SDK
// payload used for Create and Update. The Pro /v1 PUT is a full replace, so the
// complete plan representation is always sent.
func buildMobileDeviceExtensionAttributeInput(ctx context.Context, plan MobileDeviceExtensionAttributeResourceModel) (*pro.MobileDeviceExtensionAttributes, diag.Diagnostics) {
	var diags diag.Diagnostics

	ea := &pro.MobileDeviceExtensionAttributes{
		Name:                          plan.Name.ValueString(),
		DataType:                      plan.DataType.ValueString(),
		InputType:                     plan.InputType.ValueString(),
		InventoryDisplayType:          plan.InventoryDisplay.ValueString(),
		Description:                   helpers.OptionalStringPointer(plan.Description),
		LdapAttributeMapping:          helpers.OptionalStringPointer(plan.DirectoryServiceAttribute),
		LdapExtensionAttributeAllowed: helpers.OptionalBoolPointer(plan.AllowMultipleValues),
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
