// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package mobile_device_extension_attribute

import (
	datasourceTimeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/datasource/timeouts"
	resourceTimeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// MobileDeviceExtensionAttributeResourceModel is the Terraform resource model for
// a Jamf Pro mobile device extension attribute. Identical in shape to the
// computer EA minus the script-only fields (script / enabled / manage_existing_data) —
// mobile-device EAs cannot run scripts. Attribute names mirror the admin UI;
// cryptic wire names are recorded in Go comments.
//
// input_type is a discriminator: popup_menu_choices only with POPUP;
// directory_service_attribute (+ allow_multiple_values) only with
// DIRECTORY_SERVICE_ATTRIBUTE_MAPPING. inputTypeConfigValidator enforces this at
// plan time, mirroring the server's FIELD_REQUIRED / INVALID_CONTENT 400s.
type MobileDeviceExtensionAttributeResourceModel struct {
	ID               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	Description      types.String `tfsdk:"description"`
	DataType         types.String `tfsdk:"data_type"`          // wire: dataType
	InputType        types.String `tfsdk:"input_type"`         // wire: inputType
	InventoryDisplay types.String `tfsdk:"inventory_display"`  // wire: inventoryDisplayType
	PopupMenuChoices types.Set    `tfsdk:"popup_menu_choices"` // wire: popupMenuChoices (server sorts alphabetically — Set)

	// DirectoryServiceAttribute is the "Directory Service Attribute" field.
	DirectoryServiceAttribute types.String `tfsdk:"directory_service_attribute"` // wire: ldapAttributeMapping
	// AllowMultipleValues is the "Allow Multiple Values" checkbox.
	AllowMultipleValues types.Bool `tfsdk:"allow_multiple_values"` // wire: ldapExtensionAttributeAllowed

	Timeouts resourceTimeouts.Value `tfsdk:"timeouts"`
}

// MobileDeviceExtensionAttributeDataSourceModel is the Terraform data source
// model. Either id or name must be supplied (ExactlyOneOf at config validation).
type MobileDeviceExtensionAttributeDataSourceModel struct {
	ID                        types.String             `tfsdk:"id"`
	Name                      types.String             `tfsdk:"name"`
	Description               types.String             `tfsdk:"description"`
	DataType                  types.String             `tfsdk:"data_type"`
	InputType                 types.String             `tfsdk:"input_type"`
	InventoryDisplay          types.String             `tfsdk:"inventory_display"`
	PopupMenuChoices          types.Set                `tfsdk:"popup_menu_choices"`
	DirectoryServiceAttribute types.String             `tfsdk:"directory_service_attribute"`
	AllowMultipleValues       types.Bool               `tfsdk:"allow_multiple_values"`
	Timeouts                  datasourceTimeouts.Value `tfsdk:"timeouts"`
}

// mobileDeviceExtensionAttributeIdentityModel is the identity object for the
// resource and list results.
type mobileDeviceExtensionAttributeIdentityModel struct {
	ID types.String `tfsdk:"id"`
}
