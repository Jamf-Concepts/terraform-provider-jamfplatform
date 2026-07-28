// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package computer_extension_attribute

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// inputTypeConfigValidator enforces the input-type discriminator contract at
// plan time, mirroring the Jamf Pro server's per-field 400s
// (FIELD_REQUIRED / INVALID_CONTENT — wire-probed 2026-06-02):
//
//	input_type = SCRIPT                              → script REQUIRED; popup_menu_choices,
//	                                                   directory_service_attribute FORBIDDEN
//	input_type = POPUP                               → popup_menu_choices allowed (optional);
//	                                                   script, directory_service_attribute FORBIDDEN
//	input_type = DIRECTORY_SERVICE_ATTRIBUTE_MAPPING → directory_service_attribute REQUIRED;
//	                                                   script, popup_menu_choices FORBIDDEN
//	input_type = TEXT                                → script, popup_menu_choices,
//	                                                   directory_service_attribute FORBIDDEN
//	enabled = false                                  → only valid when input_type = SCRIPT
//	allow_multiple_values = true                     → only valid when input_type =
//	                                                   DIRECTORY_SERVICE_ATTRIBUTE_MAPPING
//	manage_existing_data                             → only valid when input_type = SCRIPT
//	                                                   AND enabled = false
//
// Off-the-shelf validators (ConflictsWith / AlsoRequires) fire regardless of the
// discriminator value, which is the wrong shape — the constraint is value-specific.
// Hence a custom resource.ConfigValidator (mirrors directory_binding's
// typeBlockConfigValidator). See STYLE_GUIDE §Cross-field validation.
type inputTypeConfigValidator struct{}

// Description returns a plain-text description of the validator.
func (inputTypeConfigValidator) Description(context.Context) string {
	return "the companion fields permitted for an extension attribute are keyed off input_type: SCRIPT requires script; DIRECTORY_SERVICE_ATTRIBUTE_MAPPING requires directory_service_attribute; popup_menu_choices is only valid with POPUP; only SCRIPT EAs may be disabled; allow_multiple_values is only valid with DIRECTORY_SERVICE_ATTRIBUTE_MAPPING; manage_existing_data is only valid on a disabled SCRIPT EA"
}

// MarkdownDescription returns the markdown description.
func (v inputTypeConfigValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

// ValidateResource implements the plan-time cross-field check.
func (inputTypeConfigValidator) ValidateResource(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data ComputerExtensionAttributeResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// input_type drives every rule; if it is unknown (interpolated), defer.
	if data.InputType.IsNull() || data.InputType.IsUnknown() {
		return
	}
	inputType := data.InputType.ValueString()

	scriptSet := isStringSet(data.Script)
	popupSet := !data.PopupMenuChoices.IsNull() && !data.PopupMenuChoices.IsUnknown() && len(data.PopupMenuChoices.Elements()) > 0
	dsaSet := isStringSet(data.DirectoryServiceAttribute)

	switch inputType {
	case inputTypeScript:
		if !data.Script.IsUnknown() && !scriptSet {
			resp.Diagnostics.AddAttributeError(path.Root("script"),
				"script is required when input_type = SCRIPT",
				"A SCRIPT extension attribute must supply a non-empty `script` (the script contents collected as the attribute value).")
		}
		forbid(resp, "popup_menu_choices", popupSet, inputType, inputTypePopup)
		forbid(resp, "directory_service_attribute", dsaSet, inputType, inputTypeLDAP)
	case inputTypeLDAP:
		if !data.DirectoryServiceAttribute.IsUnknown() && !dsaSet {
			resp.Diagnostics.AddAttributeError(path.Root("directory_service_attribute"),
				"directory_service_attribute is required when input_type = DIRECTORY_SERVICE_ATTRIBUTE_MAPPING",
				"A directory-service-mapped extension attribute must supply a non-empty `directory_service_attribute` (the mapped directory attribute name).")
		}
		forbid(resp, "script", scriptSet, inputType, inputTypeScript)
		forbid(resp, "popup_menu_choices", popupSet, inputType, inputTypePopup)
	case inputTypePopup:
		forbid(resp, "script", scriptSet, inputType, inputTypeScript)
		forbid(resp, "directory_service_attribute", dsaSet, inputType, inputTypeLDAP)
	case inputTypeText:
		forbid(resp, "script", scriptSet, inputType, inputTypeScript)
		forbid(resp, "popup_menu_choices", popupSet, inputType, inputTypePopup)
		forbid(resp, "directory_service_attribute", dsaSet, inputType, inputTypeLDAP)
	}

	// enabled = false is only valid for SCRIPT.
	if !data.Enabled.IsNull() && !data.Enabled.IsUnknown() && !data.Enabled.ValueBool() && inputType != inputTypeScript {
		resp.Diagnostics.AddAttributeError(path.Root("enabled"),
			"enabled may only be false when input_type = SCRIPT",
			fmt.Sprintf("Only SCRIPT extension attributes can be disabled; input_type = %q forces enabled = true. Remove `enabled = false` or set input_type = SCRIPT.", inputType))
	}

	// allow_multiple_values = true is only valid for DIRECTORY_SERVICE_ATTRIBUTE_MAPPING.
	if !data.AllowMultipleValues.IsNull() && !data.AllowMultipleValues.IsUnknown() && data.AllowMultipleValues.ValueBool() && inputType != inputTypeLDAP {
		resp.Diagnostics.AddAttributeError(path.Root("allow_multiple_values"),
			"allow_multiple_values may only be true when input_type = DIRECTORY_SERVICE_ATTRIBUTE_MAPPING",
			fmt.Sprintf("`allow_multiple_values` applies only to directory-service-mapped attributes; input_type = %q does not permit it.", inputType))
	}

	// manage_existing_data is only meaningful when a SCRIPT EA is disabled — it
	// says what to do with the inventory values already collected. Jamf Pro
	// rejects it on every other request (issue #302).
	if isStringSet(data.ManageExistingData) {
		switch {
		case inputType != inputTypeScript:
			forbid(resp, "manage_existing_data", true, inputType, inputTypeScript)
		case data.Enabled.IsUnknown():
			// Defer: enabled is interpolated, so the rule cannot be evaluated.
		case data.Enabled.IsNull() || data.Enabled.ValueBool():
			// enabled defaults to true, so an omitted enabled is an enabled EA.
			resp.Diagnostics.AddAttributeError(path.Root("manage_existing_data"),
				"manage_existing_data may only be set when enabled = false",
				"`manage_existing_data` tells Jamf Pro what to do with the inventory data already collected by a SCRIPT extension attribute when that attribute is disabled, so Jamf Pro accepts it only on an update that sets `enabled = false`. Remove it, or set `enabled = false`.")
		}
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
