// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package computer_extension_attribute

import (
	datasourceTimeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/datasource/timeouts"
	resourceTimeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ComputerExtensionAttributeResourceModel is the Terraform resource model for a
// Jamf Pro computer extension attribute. Attribute names mirror the admin UI
// (Settings → Computer management → Extension Attributes); the cryptic wire
// names are recorded in Go comments beside each field.
//
// The set of valid companion fields is keyed off input_type (a discriminator):
//   - script              valid/required only when input_type = SCRIPT
//   - popup_menu_choices   valid only when input_type = POPUP
//   - directory_service_attribute / allow_multiple_values
//     valid only when input_type = DIRECTORY_SERVICE_ATTRIBUTE_MAPPING
//   - enabled = false      valid only when input_type = SCRIPT
//
// inputTypeConfigValidator enforces all of the above at plan time, mirroring the
// server's FIELD_REQUIRED / INVALID_CONTENT 400s.
type ComputerExtensionAttributeResourceModel struct {
	ID               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	Description      types.String `tfsdk:"description"`
	DataType         types.String `tfsdk:"data_type"`          // wire: dataType
	InputType        types.String `tfsdk:"input_type"`         // wire: inputType
	InventoryDisplay types.String `tfsdk:"inventory_display"`  // wire: inventoryDisplayType
	Enabled          types.Bool   `tfsdk:"enabled"`            // wire: enabled
	Script           types.String `tfsdk:"script"`             // wire: scriptContents
	PopupMenuChoices types.Set    `tfsdk:"popup_menu_choices"` // wire: popupMenuChoices (server sorts alphabetically — Set)

	// DirectoryServiceAttribute is the "Directory Service Attribute" field.
	DirectoryServiceAttribute types.String `tfsdk:"directory_service_attribute"` // wire: ldapAttributeMapping
	// AllowMultipleValues is the "Allow Multiple Values" checkbox.
	AllowMultipleValues types.Bool `tfsdk:"allow_multiple_values"` // wire: ldapExtensionAttributeAllowed

	// ManageExistingData is a Terraform WriteOnly behavioural instruction (no
	// admin-UI field): sent on create/update to control re-collection of existing
	// inventory data for SCRIPT EAs, but never stored in state. It is null in plan
	// and state — the value is read from config in Create/Update. Jamf Pro never
	// echoes it on GET, so state never carries it.
	ManageExistingData types.String `tfsdk:"manage_existing_data"` // wire: manageExistingData (WriteOnly)

	Timeouts resourceTimeouts.Value `tfsdk:"timeouts"`
}

// ComputerExtensionAttributeDataSourceModel is the Terraform data source model.
// Either id or name must be supplied (enforced by ExactlyOneOf at config
// validation). manage_existing_data is write-only and has no data-source
// representation.
type ComputerExtensionAttributeDataSourceModel struct {
	ID                        types.String             `tfsdk:"id"`
	Name                      types.String             `tfsdk:"name"`
	Description               types.String             `tfsdk:"description"`
	DataType                  types.String             `tfsdk:"data_type"`
	InputType                 types.String             `tfsdk:"input_type"`
	InventoryDisplay          types.String             `tfsdk:"inventory_display"`
	Enabled                   types.Bool               `tfsdk:"enabled"`
	Script                    types.String             `tfsdk:"script"`
	PopupMenuChoices          types.Set                `tfsdk:"popup_menu_choices"`
	DirectoryServiceAttribute types.String             `tfsdk:"directory_service_attribute"`
	AllowMultipleValues       types.Bool               `tfsdk:"allow_multiple_values"`
	Timeouts                  datasourceTimeouts.Value `tfsdk:"timeouts"`
}

// computerExtensionAttributeIdentityModel is the identity object for the
// resource and list results.
type computerExtensionAttributeIdentityModel struct {
	ID types.String `tfsdk:"id"`
}
