// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package re_enrollment_settings

import (
	datasourceTimeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/datasource/timeouts"
	resourceTimeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ReEnrollmentSettingsResourceModel is the Terraform model for
// jamfplatform_pro_re_enrollment_settings.
//
// The resource is a singleton mapping the Jamf Pro Re-enrollment settings page:
// the five "clear …" checkboxes plus the "Clear Management History" dropdown
// that decide what device data is flushed when a previously-managed device
// re-enrolls.
type ReEnrollmentSettingsResourceModel struct {
	ID types.String `tfsdk:"id"`

	ClearPolicyLogs                 types.Bool   `tfsdk:"clear_policy_logs"`
	ClearLocationInformation        types.Bool   `tfsdk:"clear_location_information"`
	ClearLocationInformationHistory types.Bool   `tfsdk:"clear_location_information_history"`
	ClearExtensionAttributes        types.Bool   `tfsdk:"clear_extension_attributes"`
	ClearSoftwareUpdatePlans        types.Bool   `tfsdk:"clear_software_update_plans"`
	ClearManagementHistory          types.String `tfsdk:"clear_management_history"`

	Timeouts resourceTimeouts.Value `tfsdk:"timeouts"`
}

// ReEnrollmentSettingsDataSourceModel mirrors the resource model with every
// attribute Computed.
type ReEnrollmentSettingsDataSourceModel struct {
	ID types.String `tfsdk:"id"`

	ClearPolicyLogs                 types.Bool   `tfsdk:"clear_policy_logs"`
	ClearLocationInformation        types.Bool   `tfsdk:"clear_location_information"`
	ClearLocationInformationHistory types.Bool   `tfsdk:"clear_location_information_history"`
	ClearExtensionAttributes        types.Bool   `tfsdk:"clear_extension_attributes"`
	ClearSoftwareUpdatePlans        types.Bool   `tfsdk:"clear_software_update_plans"`
	ClearManagementHistory          types.String `tfsdk:"clear_management_history"`

	Timeouts datasourceTimeouts.Value `tfsdk:"timeouts"`
}

// reEnrollmentSettingsIdentityModel is the identity object used on import.
type reEnrollmentSettingsIdentityModel struct {
	ID types.String `tfsdk:"id"`
}
