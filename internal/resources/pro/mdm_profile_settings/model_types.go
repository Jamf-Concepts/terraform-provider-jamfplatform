// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package mdm_profile_settings

import (
	datasourceTimeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/datasource/timeouts"
	resourceTimeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// MDMProfileSettingsResourceModel represents the Terraform resource model for
// Jamf Pro device communication settings.
type MDMProfileSettingsResourceModel struct {
	ID                                        types.String           `tfsdk:"id"`
	AutoRenewComputerProfileWhenCaRenewed     types.Bool             `tfsdk:"auto_renew_computer_profile_when_ca_renewed"`
	AutoRenewComputerProfileBeforeExpiry      types.Bool             `tfsdk:"auto_renew_computer_profile_before_expiry"`
	ComputerProfileExpirationLimitDays        types.Int64            `tfsdk:"computer_profile_expiration_limit_days"`
	AutoRenewMobileDeviceProfileWhenCaRenewed types.Bool             `tfsdk:"auto_renew_mobile_device_profile_when_ca_renewed"`
	AutoRenewMobileDeviceProfileBeforeExpiry  types.Bool             `tfsdk:"auto_renew_mobile_device_profile_before_expiry"`
	MobileDeviceProfileExpirationLimitDays    types.Int64            `tfsdk:"mobile_device_profile_expiration_limit_days"`
	Timeouts                                  resourceTimeouts.Value `tfsdk:"timeouts"`
}

// MDMProfileSettingsDataSourceModel represents the Terraform data source model.
type MDMProfileSettingsDataSourceModel struct {
	ID                                        types.String             `tfsdk:"id"`
	AutoRenewComputerProfileWhenCaRenewed     types.Bool               `tfsdk:"auto_renew_computer_profile_when_ca_renewed"`
	AutoRenewComputerProfileBeforeExpiry      types.Bool               `tfsdk:"auto_renew_computer_profile_before_expiry"`
	ComputerProfileExpirationLimitDays        types.Int64              `tfsdk:"computer_profile_expiration_limit_days"`
	AutoRenewMobileDeviceProfileWhenCaRenewed types.Bool               `tfsdk:"auto_renew_mobile_device_profile_when_ca_renewed"`
	AutoRenewMobileDeviceProfileBeforeExpiry  types.Bool               `tfsdk:"auto_renew_mobile_device_profile_before_expiry"`
	MobileDeviceProfileExpirationLimitDays    types.Int64              `tfsdk:"mobile_device_profile_expiration_limit_days"`
	Timeouts                                  datasourceTimeouts.Value `tfsdk:"timeouts"`
}

// mdmProfileSettingsIdentityModel represents the identity object used on import.
type mdmProfileSettingsIdentityModel struct {
	ID types.String `tfsdk:"id"`
}
