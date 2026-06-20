// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package computer_check_in_settings

import (
	datasourceTimeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/datasource/timeouts"
	resourceTimeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ComputerCheckInSettingsResourceModel represents the Terraform resource model for Jamf Pro
// Client Check-In settings.
type ComputerCheckInSettingsResourceModel struct {
	ID                              types.String           `tfsdk:"id"`
	CheckInFrequency                types.Int64            `tfsdk:"check_in_frequency"`
	CreateStartupScript             types.Bool             `tfsdk:"create_startup_script"`
	StartupLog                      types.Bool             `tfsdk:"startup_log"`
	StartupPolicies                 types.Bool             `tfsdk:"startup_policies"`
	StartupSsh                      types.Bool             `tfsdk:"startup_ssh"`
	CreateLoginHook                 types.Bool             `tfsdk:"create_login_hook"`
	LoginHookLog                    types.Bool             `tfsdk:"login_hook_log"`
	LoginHookPolicies               types.Bool             `tfsdk:"login_hook_policies"`
	AllowNetworkStateChangeTriggers types.Bool             `tfsdk:"allow_network_state_change_triggers"`
	Timeouts                        resourceTimeouts.Value `tfsdk:"timeouts"`
}

// ComputerCheckInSettingsDataSourceModel represents the Terraform data source model.
type ComputerCheckInSettingsDataSourceModel struct {
	ID                              types.String             `tfsdk:"id"`
	CheckInFrequency                types.Int64              `tfsdk:"check_in_frequency"`
	CreateStartupScript             types.Bool               `tfsdk:"create_startup_script"`
	StartupLog                      types.Bool               `tfsdk:"startup_log"`
	StartupPolicies                 types.Bool               `tfsdk:"startup_policies"`
	StartupSsh                      types.Bool               `tfsdk:"startup_ssh"`
	CreateLoginHook                 types.Bool               `tfsdk:"create_login_hook"`
	LoginHookLog                    types.Bool               `tfsdk:"login_hook_log"`
	LoginHookPolicies               types.Bool               `tfsdk:"login_hook_policies"`
	AllowNetworkStateChangeTriggers types.Bool               `tfsdk:"allow_network_state_change_triggers"`
	Timeouts                        datasourceTimeouts.Value `tfsdk:"timeouts"`
}

// computerCheckInSettingsIdentityModel represents the identity object used on import.
type computerCheckInSettingsIdentityModel struct {
	ID types.String `tfsdk:"id"`
}
