// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package mobile_device_extension_attribute

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// inputTypeConfigValidator enforces the input-type discriminator contract at
// plan time, mirroring the Jamf Pro server's per-field 400s (wire-probed
// 2026-06-02). Mobile-device EAs have no SCRIPT input type:
//
//	input_type = DIRECTORY_SERVICE_ATTRIBUTE_MAPPING → directory_service_attribute REQUIRED;
//	                                                   popup_menu_choices FORBIDDEN
//	input_type = POPUP                               → popup_menu_choices allowed (optional);
//	                                                   directory_service_attribute FORBIDDEN
//	input_type = TEXT                                → popup_menu_choices,
//	                                                   directory_service_attribute FORBIDDEN
//	allow_multiple_values = true                     → only valid when input_type =
//	                                                   DIRECTORY_SERVICE_ATTRIBUTE_MAPPING
//
// Custom resource.ConfigValidator because the constraint is value-specific.
type inputTypeConfigValidator struct{}

// Description returns a plain-text description of the validator.
func (inputTypeConfigValidator) Description(context.Context) string {
	return "the companion fields permitted for a mobile-device extension attribute are keyed off input_type: DIRECTORY_SERVICE_ATTRIBUTE_MAPPING requires directory_service_attribute; popup_menu_choices is only valid with POPUP; allow_multiple_values is only valid with DIRECTORY_SERVICE_ATTRIBUTE_MAPPING"
}

// MarkdownDescription returns the markdown description.
func (v inputTypeConfigValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

// ValidateResource implements the plan-time cross-field check.
func (inputTypeConfigValidator) ValidateResource(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data MobileDeviceExtensionAttributeResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.InputType.IsNull() || data.InputType.IsUnknown() {
		return
	}
	inputType := data.InputType.ValueString()

	popupSet := !data.PopupMenuChoices.IsNull() && !data.PopupMenuChoices.IsUnknown() && len(data.PopupMenuChoices.Elements()) > 0
	dsaSet := isStringSet(data.DirectoryServiceAttribute)

	switch inputType {
	case inputTypeLDAP:
		if !data.DirectoryServiceAttribute.IsUnknown() && !dsaSet {
			resp.Diagnostics.AddAttributeError(path.Root("directory_service_attribute"),
				"directory_service_attribute is required when input_type = DIRECTORY_SERVICE_ATTRIBUTE_MAPPING",
				"A directory-service-mapped extension attribute must supply a non-empty `directory_service_attribute` (the mapped directory attribute name).")
		}
		forbid(resp, "popup_menu_choices", popupSet, inputType, inputTypePopup)
	case inputTypePopup:
		forbid(resp, "directory_service_attribute", dsaSet, inputType, inputTypeLDAP)
	case inputTypeText:
		forbid(resp, "popup_menu_choices", popupSet, inputType, inputTypePopup)
		forbid(resp, "directory_service_attribute", dsaSet, inputType, inputTypeLDAP)
	}

	// allow_multiple_values = true is only valid for DIRECTORY_SERVICE_ATTRIBUTE_MAPPING.
	if !data.AllowMultipleValues.IsNull() && !data.AllowMultipleValues.IsUnknown() && data.AllowMultipleValues.ValueBool() && inputType != inputTypeLDAP {
		resp.Diagnostics.AddAttributeError(path.Root("allow_multiple_values"),
			"allow_multiple_values may only be true when input_type = DIRECTORY_SERVICE_ATTRIBUTE_MAPPING",
			fmt.Sprintf("`allow_multiple_values` applies only to directory-service-mapped attributes; input_type = %q does not permit it.", inputType))
	}
}

// isStringSet reports whether a string attribute carries a usable (known,
// non-null, non-empty) value at config time.
func isStringSet(s types.String) bool {
	return !s.IsNull() && !s.IsUnknown() && s.ValueString() != ""
}

// forbid adds a "field forbidden for this input_type" diagnostic when set is
// true. allowedType names the only input_type for which the field is valid.
func forbid(resp *resource.ValidateConfigResponse, attr string, set bool, inputType, allowedType string) {
	if !set {
		return
	}
	resp.Diagnostics.AddAttributeError(path.Root(attr),
		fmt.Sprintf("%s is not valid when input_type = %s", attr, inputType),
		fmt.Sprintf("`%s` may only be set when input_type = %s. Remove it, or change input_type to %s.", attr, allowedType, allowedType))
}

// Compile-time interface assertion.
var _ resource.ConfigValidator = inputTypeConfigValidator{}
