// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package access_management_settings

import (
	datasourceTimeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/datasource/timeouts"
	resourceTimeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// AccessManagementSettingsResourceModel represents the Terraform resource model for
// Jamf Pro Access Management settings.
type AccessManagementSettingsResourceModel struct {
	ID                                  types.String           `tfsdk:"id"`
	AutomatedDeviceEnrollmentServerUUID types.String           `tfsdk:"automated_device_enrollment_server_uuid"`
	Timeouts                            resourceTimeouts.Value `tfsdk:"timeouts"`
}

// AccessManagementSettingsDataSourceModel represents the Terraform data source model.
type AccessManagementSettingsDataSourceModel struct {
	ID                                  types.String             `tfsdk:"id"`
	AutomatedDeviceEnrollmentServerUUID types.String             `tfsdk:"automated_device_enrollment_server_uuid"`
	Timeouts                            datasourceTimeouts.Value `tfsdk:"timeouts"`
}

// accessManagementSettingsIdentityModel represents the identity object used on import.
type accessManagementSettingsIdentityModel struct {
	ID types.String `tfsdk:"id"`
}
