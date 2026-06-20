// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package app_installer_settings

import (
	datasourceTimeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/datasource/timeouts"
	resourceTimeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// DeploymentSettingsModel holds the deployment_settings sub-block.
type DeploymentSettingsModel struct {
	BatchSize      types.Int64  `tfsdk:"batch_size"`
	BatchFrequency types.Int64  `tfsdk:"batch_frequency"`
	Days           types.Set    `tfsdk:"days"`
	ServerTimeFrom types.String `tfsdk:"server_time_from"`
	ServerTimeTo   types.String `tfsdk:"server_time_to"`
}

// EndUserExperienceModel holds the end_user_experience sub-block.
type EndUserExperienceModel struct {
	NotificationFrequency types.Int64  `tfsdk:"notification_frequency"`
	NotificationMessage   types.String `tfsdk:"notification_message"`
	UpdateDeadline        types.Int64  `tfsdk:"update_deadline"`
	ForceQuitMessage      types.String `tfsdk:"force_quit_message"`
	ForceQuitGracePeriod  types.Int64  `tfsdk:"force_quit_grace_period"`
	UpdateCompleteMessage types.String `tfsdk:"update_complete_message"`
	Relaunch              types.Bool   `tfsdk:"relaunch"`
	Suppress              types.Bool   `tfsdk:"suppress"`
}

// AppInstallerSettingsResourceModel is the Terraform resource model.
//
// deployment_settings and end_user_experience are types.Object (not *struct
// pointers) because they are Optional+Computed: the framework sets an Unknown
// value in the plan when the user omits the block, and *struct cannot hold
// Unknown — see [[feedback_optional_computed_nested_object]].
type AppInstallerSettingsResourceModel struct {
	ID                 types.String           `tfsdk:"id"`
	DeploymentSettings types.Object           `tfsdk:"deployment_settings"`
	EndUserExperience  types.Object           `tfsdk:"end_user_experience"`
	Timeouts           resourceTimeouts.Value `tfsdk:"timeouts"`
}

// AppInstallerSettingsDataSourceModel is the Terraform data source model.
// Blocks are types.Object for consistency with the resource model.
type AppInstallerSettingsDataSourceModel struct {
	ID                 types.String             `tfsdk:"id"`
	DeploymentSettings types.Object             `tfsdk:"deployment_settings"`
	EndUserExperience  types.Object             `tfsdk:"end_user_experience"`
	Timeouts           datasourceTimeouts.Value `tfsdk:"timeouts"`
}

// appInstallerSettingsIdentityModel is the identity object used on import.
type appInstallerSettingsIdentityModel struct {
	ID types.String `tfsdk:"id"`
}
