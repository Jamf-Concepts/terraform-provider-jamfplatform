// Copyright 2025 Jamf Software LLC.

package device

import (
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// DeviceDataSource implements the Terraform data source for Jamf devices.
type DeviceDataSource struct {
	client *client.Client
}

// DeviceDataSourceModel represents the Terraform data source model for a Jamf device lookup.
type DeviceDataSourceModel struct {
	ID                     types.String `tfsdk:"id"`
	SerialNumber           types.String `tfsdk:"serial_number"`
	Name                   types.String `tfsdk:"name"`
	Model                  types.String `tfsdk:"model"`
	ModelIdentifier        types.String `tfsdk:"model_identifier"`
	LastInventoryUpdate    types.String `tfsdk:"last_inventory_update_time"`
	LastCheckIn            types.String `tfsdk:"last_check_in_time"`
	OperatingSystemVersion types.String `tfsdk:"operating_system_version"`
	UserID                 types.String `tfsdk:"user_id"`
	EnrollmentType         types.String `tfsdk:"enrollment_type"`
	LastEnrollmentTime     types.String `tfsdk:"last_enrollment_time"`
}
