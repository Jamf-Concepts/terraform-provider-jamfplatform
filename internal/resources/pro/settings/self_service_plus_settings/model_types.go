// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package self_service_plus_settings

import (
	datasourceTimeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/datasource/timeouts"
	resourceTimeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// SelfServicePlusSettingsResourceModel represents the Terraform resource model for
// Jamf Pro Self Service Plus settings.
type SelfServicePlusSettingsResourceModel struct {
	ID       types.String           `tfsdk:"id"`
	Enabled  types.Bool             `tfsdk:"enabled"`
	Timeouts resourceTimeouts.Value `tfsdk:"timeouts"`
}

// SelfServicePlusSettingsDataSourceModel represents the Terraform data source model.
type SelfServicePlusSettingsDataSourceModel struct {
	ID       types.String             `tfsdk:"id"`
	Enabled  types.Bool               `tfsdk:"enabled"`
	Timeouts datasourceTimeouts.Value `tfsdk:"timeouts"`
}

// selfServicePlusSettingsIdentityModel represents the identity object used on import.
type selfServicePlusSettingsIdentityModel struct {
	ID types.String `tfsdk:"id"`
}
