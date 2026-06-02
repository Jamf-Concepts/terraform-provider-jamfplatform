// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package mobile_device_extension_attribute

import (
	"context"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/pro"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
)

// assignMobileDeviceExtensionAttributeResourceModel populates a resource model
// from a MobileDeviceExtensionAttributes response. Always sourced from a GET
// after Create/Update. Jamf Pro echoes inapplicable companion fields as empty
// values ("" / [] / false) rather than null, so user-authored string fields are
// reconciled to collapse the empty echo back to null.
func assignMobileDeviceExtensionAttributeResourceModel(ctx context.Context, state *MobileDeviceExtensionAttributeResourceModel, ea *pro.MobileDeviceExtensionAttributes) diag.Diagnostics {
	var diags diag.Diagnostics
	if ea == nil {
		return diags
	}

	state.ID = helpers.StringPointerValueOrNull(ea.ID)
	state.Name = types.StringValue(ea.Name)
	state.Description = helpers.ReconcileOptionalStringPointer(ea.Description, state.Description)
	state.DataType = types.StringValue(ea.DataType)
	state.InputType = types.StringValue(ea.InputType)
	state.InventoryDisplay = types.StringValue(ea.InventoryDisplayType)
	state.DirectoryServiceAttribute = helpers.ReconcileOptionalStringPointer(ea.LdapAttributeMapping, state.DirectoryServiceAttribute)
	state.AllowMultipleValues = helpers.BoolPointerValueOrNull(ea.LdapExtensionAttributeAllowed)

	choices, choicesDiags := flattenPopupMenuChoices(ctx, ea.PopupMenuChoices)
	diags.Append(choicesDiags...)
	if diags.HasError() {
		return diags
	}
	state.PopupMenuChoices = choices

	return diags
}

// assignMobileDeviceExtensionAttributeDataSourceModel populates a data source model.
func assignMobileDeviceExtensionAttributeDataSourceModel(ctx context.Context, state *MobileDeviceExtensionAttributeDataSourceModel, ea *pro.MobileDeviceExtensionAttributes) diag.Diagnostics {
	var diags diag.Diagnostics
	if ea == nil {
		return diags
	}

	state.ID = helpers.StringPointerValueOrNull(ea.ID)
	state.Name = types.StringValue(ea.Name)
	state.Description = helpers.StringPointerValueOrNull(ea.Description)
	state.DataType = types.StringValue(ea.DataType)
	state.InputType = types.StringValue(ea.InputType)
	state.InventoryDisplay = types.StringValue(ea.InventoryDisplayType)
	state.DirectoryServiceAttribute = helpers.StringPointerValueOrNull(ea.LdapAttributeMapping)
	state.AllowMultipleValues = helpers.BoolPointerValueOrNull(ea.LdapExtensionAttributeAllowed)

	choices, choicesDiags := flattenPopupMenuChoices(ctx, ea.PopupMenuChoices)
	diags.Append(choicesDiags...)
	if diags.HasError() {
		return diags
	}
	state.PopupMenuChoices = choices

	return diags
}

// flattenPopupMenuChoices converts the SDK popup-choices slice into a Set of
// strings. Returns a null Set for an empty or absent slice. Modelled as a Set
// because Jamf Pro returns the choices sorted alphabetically, not in submitted
// order — a List would perpetually diff after apply.
func flattenPopupMenuChoices(ctx context.Context, src *[]string) (types.Set, diag.Diagnostics) {
	var diags diag.Diagnostics
	if src == nil || len(*src) == 0 {
		return types.SetNull(types.StringType), diags
	}
	set, d := types.SetValueFrom(ctx, types.StringType, *src)
	diags.Append(d...)
	return set, diags
}
