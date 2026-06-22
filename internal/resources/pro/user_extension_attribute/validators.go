// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package user_extension_attribute

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// inputTypeConfigValidator enforces that popup_menu_choices is only supplied when
// input_type = "Pop-up Menu". The Classic API does not enforce this server-side
// (it is permissive), so the check is purely client-side to keep configs honest.
type inputTypeConfigValidator struct{}

// Description returns a plain-text description of the validator.
func (inputTypeConfigValidator) Description(context.Context) string {
	return "popup_menu_choices may only be set when input_type = \"Pop-up Menu\""
}

// MarkdownDescription returns the markdown description.
func (v inputTypeConfigValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

// ValidateResource implements the plan-time cross-field check.
func (inputTypeConfigValidator) ValidateResource(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data UserExtensionAttributeResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.InputType.IsNull() || data.InputType.IsUnknown() {
		return
	}

	popupSet := !data.PopupMenuChoices.IsNull() && !data.PopupMenuChoices.IsUnknown() && len(data.PopupMenuChoices.Elements()) > 0
	if popupSet && data.InputType.ValueString() != inputTypePopupMenu {
		resp.Diagnostics.AddAttributeError(path.Root("popup_menu_choices"),
			"popup_menu_choices is not valid for this input_type",
			"`popup_menu_choices` may only be set when input_type = \"Pop-up Menu\". Remove it, or change input_type to \"Pop-up Menu\".")
	}
}

// Compile-time interface assertion.
var _ resource.ConfigValidator = inputTypeConfigValidator{}
