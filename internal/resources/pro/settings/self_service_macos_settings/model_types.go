// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package self_service_macos_settings

import (
	datasourceTimeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/datasource/timeouts"
	resourceTimeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// SelfServiceMacosSettingsResourceModel represents the Terraform resource model for the
// Jamf Pro Self Service for macOS app settings.
type SelfServiceMacosSettingsResourceModel struct {
	ID                               types.String           `tfsdk:"id"`
	InstallAutomatically             types.Bool             `tfsdk:"install_automatically"`
	InstallLocation                  types.String           `tfsdk:"install_location"`
	LoginMethod                      types.String           `tfsdk:"login_method"`
	AuthenticationType               types.String           `tfsdk:"authentication_type"`
	KeychainCredentialStorageEnabled types.Bool             `tfsdk:"keychain_credential_storage_enabled"`
	Fido2Enabled                     types.Bool             `tfsdk:"fido2_enabled"`
	NotificationsEnabled             types.Bool             `tfsdk:"notifications_enabled"`
	AlertUserApprovedMdm             types.Bool             `tfsdk:"alert_user_approved_mdm"`
	DefaultLandingPage               types.String           `tfsdk:"default_landing_page"`
	DefaultHomeCategoryID            types.Int64            `tfsdk:"default_home_category_id"`
	BookmarksDisplayName             types.String           `tfsdk:"bookmarks_display_name"`
	Timeouts                         resourceTimeouts.Value `tfsdk:"timeouts"`
}

// SelfServiceMacosSettingsDataSourceModel represents the Terraform data source model.
type SelfServiceMacosSettingsDataSourceModel struct {
	ID                               types.String             `tfsdk:"id"`
	InstallAutomatically             types.Bool               `tfsdk:"install_automatically"`
	InstallLocation                  types.String             `tfsdk:"install_location"`
	LoginMethod                      types.String             `tfsdk:"login_method"`
	AuthenticationType               types.String             `tfsdk:"authentication_type"`
	KeychainCredentialStorageEnabled types.Bool               `tfsdk:"keychain_credential_storage_enabled"`
	Fido2Enabled                     types.Bool               `tfsdk:"fido2_enabled"`
	NotificationsEnabled             types.Bool               `tfsdk:"notifications_enabled"`
	AlertUserApprovedMdm             types.Bool               `tfsdk:"alert_user_approved_mdm"`
	DefaultLandingPage               types.String             `tfsdk:"default_landing_page"`
	DefaultHomeCategoryID            types.Int64              `tfsdk:"default_home_category_id"`
	BookmarksDisplayName             types.String             `tfsdk:"bookmarks_display_name"`
	Timeouts                         datasourceTimeouts.Value `tfsdk:"timeouts"`
}

// selfServiceMacosSettingsIdentityModel represents the identity object used on import.
type selfServiceMacosSettingsIdentityModel struct {
	ID types.String `tfsdk:"id"`
}
