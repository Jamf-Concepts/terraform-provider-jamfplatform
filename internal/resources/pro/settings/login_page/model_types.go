// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package login_page

import (
	datasourceTimeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/datasource/timeouts"
	resourceTimeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// LoginPageSettingsResourceModel represents the Terraform resource model for the
// Jamf Pro login page (login-customization) settings.
type LoginPageSettingsResourceModel struct {
	ID                      types.String           `tfsdk:"id"`
	IncludeCustomDisclaimer types.Bool             `tfsdk:"include_custom_disclaimer"`
	DisclaimerHeading       types.String           `tfsdk:"disclaimer_heading"`
	DisclaimerMainText      types.String           `tfsdk:"disclaimer_main_text"`
	ActionText              types.String           `tfsdk:"action_text"`
	Timeouts                resourceTimeouts.Value `tfsdk:"timeouts"`
}

// LoginPageSettingsDataSourceModel represents the Terraform data source model.
type LoginPageSettingsDataSourceModel struct {
	ID                      types.String             `tfsdk:"id"`
	IncludeCustomDisclaimer types.Bool               `tfsdk:"include_custom_disclaimer"`
	DisclaimerHeading       types.String             `tfsdk:"disclaimer_heading"`
	DisclaimerMainText      types.String             `tfsdk:"disclaimer_main_text"`
	ActionText              types.String             `tfsdk:"action_text"`
	Timeouts                datasourceTimeouts.Value `tfsdk:"timeouts"`
}

// loginPageSettingsIdentityModel represents the identity object used on import.
type loginPageSettingsIdentityModel struct {
	ID types.String `tfsdk:"id"`
}
