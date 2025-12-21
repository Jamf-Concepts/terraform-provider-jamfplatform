// Copyright 2025 Jamf Software LLC.

package device

import (
	"context"
	"fmt"
	"time"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/datasource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const defaultDeviceReadTimeout = 30 * time.Second

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &DeviceDataSource{}

// NewDeviceDataSource returns a new instance of DeviceDataSource.
func NewDeviceDataSource() datasource.DataSource {
	return &DeviceDataSource{}
}

// Metadata sets the data source type name for the Terraform provider.
func (d *DeviceDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_device"
}

// Schema defines the data source schema.
func (d *DeviceDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lookup a Jamf device by ID via the Device Inventory API. Requires **Device Inventory API** access.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Device UUID (Jamf Pro Management ID) to query.",
				Required:            true,
			},
			"serial_number": schema.StringAttribute{
				MarkdownDescription: "Device serial number reported by inventory (case-sensitive).",
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Device name reported by inventory.",
				Computed:            true,
			},
			"model": schema.StringAttribute{
				MarkdownDescription: "Marketing model string.",
				Computed:            true,
			},
			"model_identifier": schema.StringAttribute{
				MarkdownDescription: "Model identifier (e.g., Mac14,6).",
				Computed:            true,
			},
			"last_inventory_update_time": schema.StringAttribute{
				MarkdownDescription: "Timestamp of the last inventory update in ISO 8601 format.",
				Computed:            true,
			},
			"last_check_in_time": schema.StringAttribute{
				MarkdownDescription: "Timestamp of the last check-in in ISO 8601 format.",
				Computed:            true,
			},
			"operating_system_version": schema.StringAttribute{
				MarkdownDescription: "Operating system version string.",
				Computed:            true,
			},
			"user_id": schema.StringAttribute{
				MarkdownDescription: "User ID associated with the device, if any.",
				Computed:            true,
			},
			"enrollment_type": schema.StringAttribute{
				MarkdownDescription: "Enrollment type reported by the API.",
				Computed:            true,
			},
			"last_enrollment_time": schema.StringAttribute{
				MarkdownDescription: "Timestamp of the last enrollment in ISO 8601 format.",
				Computed:            true,
			},
			"managed": schema.BoolAttribute{
				MarkdownDescription: "Indicates whether the device is managed.",
				Computed:            true,
			},
			"supervised": schema.BoolAttribute{
				MarkdownDescription: "Indicates whether the device is supervised.",
				Computed:            true,
			},
			"hardware_make": schema.StringAttribute{
				MarkdownDescription: "Hardware make from the detailed device record.",
				Computed:            true,
			},
			"hardware_udid": schema.StringAttribute{
				MarkdownDescription: "Hardware UDID reported by inventory.",
				Computed:            true,
			},
			"hardware_battery_health": schema.StringAttribute{
				MarkdownDescription: "Battery health reported by inventory.",
				Computed:            true,
			},
			"hardware_mac_address": schema.StringAttribute{
				MarkdownDescription: "Primary hardware MAC address.",
				Computed:            true,
			},
			"hardware_storage_capacity": schema.Int64Attribute{
				MarkdownDescription: "Total storage capacity in bytes.",
				Computed:            true,
			},
			"hardware_storage_used": schema.Int64Attribute{
				MarkdownDescription: "Used storage in bytes.",
				Computed:            true,
			},
			"network_last_ip_address": schema.StringAttribute{
				MarkdownDescription: "Last IP address reported for the device.",
				Computed:            true,
			},
			"network_last_reported_ipv4_address": schema.StringAttribute{
				MarkdownDescription: "Last reported IPv4 address.",
				Computed:            true,
			},
			"network_last_reported_ipv6_address": schema.StringAttribute{
				MarkdownDescription: "Last reported IPv6 address.",
				Computed:            true,
			},
			"operating_system_name": schema.StringAttribute{
				MarkdownDescription: "Operating system name.",
				Computed:            true,
			},
			"operating_system_build": schema.StringAttribute{
				MarkdownDescription: "Operating system build string.",
				Computed:            true,
			},
			"operating_system_supplemental_build_version": schema.StringAttribute{
				MarkdownDescription: "Supplemental build version, if provided.",
				Computed:            true,
			},
			"operating_system_rapid_security_response": schema.StringAttribute{
				MarkdownDescription: "Rapid Security Response build identifier, if provided.",
				Computed:            true,
			},
			"security_bootstrap_token_escrowed_status": schema.StringAttribute{
				MarkdownDescription: "Bootstrap token escrow status.",
				Computed:            true,
			},
			"security_hardware_encryption": schema.BoolAttribute{
				MarkdownDescription: "Indicates whether hardware encryption is enabled.",
				Computed:            true,
			},
			"security_passcode_present": schema.BoolAttribute{
				MarkdownDescription: "Indicates whether a passcode is present.",
				Computed:            true,
			},
			"security_passcode_compliant": schema.BoolAttribute{
				MarkdownDescription: "Indicates whether the passcode complies with policy.",
				Computed:            true,
			},
			"security_lost_mode_enabled": schema.BoolAttribute{
				MarkdownDescription: "Indicates whether lost mode is enabled.",
				Computed:            true,
			},
			"timeouts": timeouts.Attributes(ctx),
		},
	}
}

// Configure sets up the API client for the data source from the provider configuration.
func (d *DeviceDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = client
}

// Read fetches a device by ID and populates the Terraform state.
func (d *DeviceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data DeviceDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if d.client == nil {
		resp.Diagnostics.AddError(
			"Provider not configured",
			"The provider client was not configured. Please ensure the provider block is set up correctly.",
		)
		return
	}

	lookupID := ""
	if !data.ID.IsNull() && !data.ID.IsUnknown() {
		lookupID = data.ID.ValueString()
	}
	if lookupID == "" {
		resp.Diagnostics.AddError(
			"Missing device id",
			"The id attribute must be provided to read a device.",
		)
		return
	}

	readTimeout := defaultDeviceReadTimeout
	if !data.Timeouts.IsNull() && !data.Timeouts.IsUnknown() {
		configuredTimeout, timeoutDiags := data.Timeouts.Read(ctx, defaultDeviceReadTimeout)
		resp.Diagnostics.Append(timeoutDiags...)
		if resp.Diagnostics.HasError() {
			return
		}
		readTimeout = configuredTimeout
	}

	readCtx, cancel := context.WithTimeout(ctx, readTimeout)
	defer cancel()

	deviceDetail, err := d.client.GetDeviceByIDV1(readCtx, lookupID)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read device", fmt.Sprintf("Failed to get device %s: %s", lookupID, err))
		return
	}

	serialValue := ""
	if deviceDetail.Hardware != nil && deviceDetail.Hardware.SerialNumber != "" {
		serialValue = deviceDetail.Hardware.SerialNumber
	}

	modelValue := ""
	if deviceDetail.Hardware != nil && deviceDetail.Hardware.Model != "" {
		modelValue = deviceDetail.Hardware.Model
	}

	modelIdentifierValue := ""
	if deviceDetail.Hardware != nil && deviceDetail.Hardware.ModelIdentifier != "" {
		modelIdentifierValue = deviceDetail.Hardware.ModelIdentifier
	}

	operatingSystemVersion := ""
	if deviceDetail.OperatingSystem != nil && deviceDetail.OperatingSystem.Version != "" {
		operatingSystemVersion = deviceDetail.OperatingSystem.Version
	}

	data.ID = types.StringValue(deviceDetail.ID)
	data.SerialNumber = stringValueOrNull(serialValue)
	data.Name = stringValueOrNull(deviceDetail.Name)
	data.Model = stringValueOrNull(modelValue)
	data.ModelIdentifier = stringValueOrNull(modelIdentifierValue)
	data.LastInventoryUpdate = stringValueOrNull(deviceDetail.LastInventoryUpdateTime)
	data.LastCheckIn = stringPointerValueOrNull(deviceDetail.LastCheckInTime)
	data.OperatingSystemVersion = stringValueOrNull(operatingSystemVersion)
	if deviceDetail.OperatingSystem != nil {
		data.OperatingSystemName = stringValueOrNull(deviceDetail.OperatingSystem.Name)
		data.OperatingSystemBuild = stringValueOrNull(deviceDetail.OperatingSystem.Build)
		data.OperatingSystemSupplementalBuildVersion = stringPointerValueOrNull(deviceDetail.OperatingSystem.SupplementalBuildVersion)
		data.OperatingSystemRapidSecurityResponse = stringPointerValueOrNull(deviceDetail.OperatingSystem.RapidSecurityResponse)
	} else {
		data.OperatingSystemName = types.StringNull()
		data.OperatingSystemBuild = types.StringNull()
		data.OperatingSystemSupplementalBuildVersion = types.StringNull()
		data.OperatingSystemRapidSecurityResponse = types.StringNull()
	}
	data.UserID = stringPointerValueOrNull(deviceDetail.UserID)
	data.EnrollmentType = stringValueOrNull(deviceDetail.EnrollmentType)
	data.LastEnrollmentTime = stringValueOrNull(deviceDetail.LastEnrollmentTime)
	data.Managed = types.BoolValue(deviceDetail.Managed)
	data.Supervised = types.BoolValue(deviceDetail.Supervised)

	if deviceDetail.Hardware != nil {
		data.HardwareMake = stringValueOrNull(deviceDetail.Hardware.Make)
		data.HardwareUDID = stringValueOrNull(deviceDetail.Hardware.UDID)
		data.HardwareBatteryHealth = stringValueOrNull(deviceDetail.Hardware.BatteryHealth)
		data.HardwareMacAddress = stringValueOrNull(deviceDetail.Hardware.MacAddress)
		data.HardwareStorageCapacity = types.Int64Value(int64(deviceDetail.Hardware.StorageCapacity))
		data.HardwareStorageUsed = types.Int64Value(int64(deviceDetail.Hardware.StorageUsed))
	} else {
		data.HardwareMake = types.StringNull()
		data.HardwareUDID = types.StringNull()
		data.HardwareBatteryHealth = types.StringNull()
		data.HardwareMacAddress = types.StringNull()
		data.HardwareStorageCapacity = types.Int64Null()
		data.HardwareStorageUsed = types.Int64Null()
	}

	if deviceDetail.Network != nil {
		data.NetworkLastIPAddress = stringPointerValueOrNull(deviceDetail.Network.LastIPAddress)
		data.NetworkLastReportedIPv4Address = stringPointerValueOrNull(deviceDetail.Network.LastReportedIPv4Address)
		data.NetworkLastReportedIPv6Address = stringPointerValueOrNull(deviceDetail.Network.LastReportedIPv6Address)
	} else {
		data.NetworkLastIPAddress = types.StringNull()
		data.NetworkLastReportedIPv4Address = types.StringNull()
		data.NetworkLastReportedIPv6Address = types.StringNull()
	}

	if deviceDetail.Security != nil {
		data.SecurityBootstrapTokenEscrowedStatus = stringValueOrNull(deviceDetail.Security.BootstrapTokenEscrowedStatus)
		data.SecurityHardwareEncryption = boolPointerValueOrNull(deviceDetail.Security.HardwareEncryption)
		data.SecurityPasscodePresent = boolPointerValueOrNull(deviceDetail.Security.PasscodePresent)
		data.SecurityPasscodeCompliant = boolPointerValueOrNull(deviceDetail.Security.PasscodeCompliant)
		data.SecurityLostModeEnabled = boolPointerValueOrNull(deviceDetail.Security.LostModeEnabled)
	} else {
		data.SecurityBootstrapTokenEscrowedStatus = types.StringNull()
		data.SecurityHardwareEncryption = types.BoolNull()
		data.SecurityPasscodePresent = types.BoolNull()
		data.SecurityPasscodeCompliant = types.BoolNull()
		data.SecurityLostModeEnabled = types.BoolNull()
	}

	tflog.Trace(ctx, "read device data source", map[string]interface{}{
		"id": deviceDetail.ID,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
