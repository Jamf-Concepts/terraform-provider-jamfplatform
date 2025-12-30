// Copyright 2025 Jamf Software LLC.

package mobiledevice

import (
	"context"
	"fmt"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/client"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &DataSourceMobileDevice{}

// NewDataSourceMobileDevice returns a new instance of DataSourceMobileDevice.
func NewDataSourceMobileDevice() datasource.DataSource {
	return &DataSourceMobileDevice{}
}

// Metadata sets the data source type name for the Terraform provider.
func (d *DataSourceMobileDevice) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_inventory_mobile_device"
}

// Schema defines the schema for the mobile device data source.
func (d *DataSourceMobileDevice) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lookup a Jamf mobile device by ID from an Environment via the Mobile Device Inventory API. Requires **Inventory API** access.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The ID of the mobile device to retrieve.",
			},
			"mobile_device_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Mobile device ID from the API response.",
			},
			"device_type": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Type of the device.",
			},
			"sections": schema.ListAttribute{
				ElementType:         types.StringType,
				Optional:            true,
				MarkdownDescription: "Sections to retrieve (e.g., `GENERAL`, `HARDWARE`, `SECURITY`). If not specified, all sections are retrieved.",
			},
			"udid": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "UDID of the device.",
			},
			"display_name": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Display name of the device.",
			},
			"asset_tag": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Asset tag.",
			},
			"site_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Site ID.",
			},
			"last_inventory_update_date": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Last inventory update date.",
			},
			"os_version": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "OS version.",
			},
			"os_build": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "OS build.",
			},
			"ip_address": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "IP address.",
			},
			"managed": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the device is managed.",
			},
			"supervised": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the device is supervised.",
			},
			"device_ownership_type": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Device ownership type.",
			},
			"last_enrolled_date": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Last enrolled date.",
			},
			"mdm_profile_expiration": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "MDM profile expiration date.",
			},
			"time_zone": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Device time zone.",
			},
			"capacity_mb": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Storage capacity in MB.",
			},
			"available_space_mb": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Available space in MB.",
			},
			"used_space_percentage": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Used space percentage.",
			},
			"battery_level": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Battery level percentage.",
			},
			"battery_health": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Battery health status.",
			},
			"serial_number": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Serial number.",
			},
			"hardware_wifi_mac_address": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "WiFi MAC address from hardware section.",
			},
			"bluetooth_mac_address": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Bluetooth MAC address.",
			},
			"model": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Device model.",
			},
			"model_identifier": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Model identifier.",
			},
			"model_number": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Model number.",
			},
			"device_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Device ID.",
			},
			"username": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Username.",
			},
			"real_name": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Real name.",
			},
			"email_address": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Email address.",
			},
			"position": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Position.",
			},
			"phone_number": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Phone number.",
			},
			"department_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Department ID.",
			},
			"building_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Building ID.",
			},
			"room": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Room.",
			},
			"building": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Building name.",
			},
			"department": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Department name.",
			},
			"purchased": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether purchased.",
			},
			"leased": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether leased.",
			},
			"po_number": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Purchase order number.",
			},
			"vendor": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Vendor.",
			},
			"apple_care_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "AppleCare ID.",
			},
			"purchase_price": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Purchase price.",
			},
			"purchasing_account": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Purchasing account.",
			},
			"po_date": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Purchase order date.",
			},
			"warranty_expires_date": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Warranty expiration date.",
			},
			"lease_expires_date": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Lease expiration date.",
			},
			"life_expectancy": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Life expectancy in years.",
			},
			"purchasing_contact": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Purchasing contact.",
			},
			"data_protected": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether data is protected.",
			},
			"block_level_encryption_capable": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether block level encryption capable.",
			},
			"file_level_encryption_capable": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether file level encryption capable.",
			},
			"passcode_present": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether passcode is present.",
			},
			"passcode_compliant": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether passcode is compliant.",
			},
			"activation_lock_enabled": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether activation lock is enabled.",
			},
			"jail_break_detected": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether jailbreak is detected.",
			},
			"lost_mode_enabled": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether lost mode is enabled.",
			},
			"lost_mode_message": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Lost mode message.",
			},
			"lost_mode_phone_number": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Lost mode phone number.",
			},
			"cellular_technology": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Cellular technology.",
			},
			"iccid": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "ICCID.",
			},
			"carrier": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Carrier.",
			},
			"sim_phone_number": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "SIM phone number.",
			},
			"network_wifi_mac_address": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "WiFi MAC address from network section.",
			},
			"bluetooth_mac": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Bluetooth MAC address.",
			},
			"ethernet_mac": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Ethernet MAC address.",
			},
			"applications": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Applications installed on the device.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"identifier": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Application identifier.",
						},
						"name": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Application name.",
						},
						"version": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Application version.",
						},
						"short_version": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Short version.",
						},
						"management_status": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Management status.",
						},
						"bundle_size": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Bundle size.",
						},
						"dynamic_size": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Dynamic size.",
						},
					},
				},
			},
			"profiles": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Configuration profiles installed on the device.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"display_name": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Display name.",
						},
						"version": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Profile version.",
						},
						"uuid": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Profile UUID.",
						},
						"identifier": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Profile identifier.",
						},
						"removable": schema.BoolAttribute{
							Computed:            true,
							MarkdownDescription: "Whether removable.",
						},
						"last_installed": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Last installed date.",
						},
						"username": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Username.",
						},
					},
				},
			},
			"certificates": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Certificates installed on the device.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"common_name": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Common name.",
						},
						"identity": schema.BoolAttribute{
							Computed:            true,
							MarkdownDescription: "Whether identity certificate.",
						},
						"expiration_date": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Expiration date.",
						},
					},
				},
			},
		},
	}
}

// Configure sets up the API client for the data source from the provider configuration.
func (d *DataSourceMobileDevice) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

// Read fetches the mobile device details and sets the state.
func (d *DataSourceMobileDevice) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data MobileDeviceDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	id := data.ID.ValueString()
	if id == "" {
		resp.Diagnostics.AddError(
			"Missing mobile device ID",
			"Mobile device ID is required to retrieve device details.",
		)
		return
	}

	var sections []string
	if helpers.IsConfiguredValue(data.Sections) {
		resp.Diagnostics.Append(data.Sections.ElementsAs(ctx, &sections, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	mobileDevice, err := d.client.GetInventoryMobileDeviceByIDV1(ctx, id, sections)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to get mobile device",
			fmt.Sprintf("Error retrieving mobile device with ID %s: %s", id, err),
		)
		return
	}

	data.ID = types.StringValue(id)
	data.MobileDeviceId = types.StringValue(mobileDevice.MobileDeviceId)
	data.DeviceType = types.StringValue(mobileDevice.DeviceType)
	data.Udid = types.StringValue(mobileDevice.General.Udid)
	data.DisplayName = types.StringValue(mobileDevice.General.DisplayName)
	data.AssetTag = types.StringValue(mobileDevice.General.AssetTag)
	data.SiteId = types.StringValue(mobileDevice.General.SiteId)
	data.LastInventoryUpdateDate = types.StringValue(mobileDevice.General.LastInventoryUpdateDate)
	data.OsVersion = types.StringValue(mobileDevice.General.OsVersion)
	data.OsBuild = types.StringValue(mobileDevice.General.OsBuild)
	data.IpAddress = types.StringValue(mobileDevice.General.IpAddress)
	data.Managed = types.BoolValue(mobileDevice.General.Managed)
	data.Supervised = types.BoolValue(mobileDevice.General.Supervised)
	data.DeviceOwnershipType = types.StringValue(mobileDevice.General.DeviceOwnershipType)
	data.LastEnrolledDate = types.StringValue(mobileDevice.General.LastEnrolledDate)
	data.MdmProfileExpiration = types.StringValue(mobileDevice.General.MdmProfileExpiration)
	data.TimeZone = types.StringValue(mobileDevice.General.TimeZone)
	data.CapacityMb = types.Int64Value(int64(mobileDevice.Hardware.CapacityMb))
	data.AvailableSpaceMb = types.Int64Value(int64(mobileDevice.Hardware.AvailableSpaceMb))
	data.UsedSpacePercentage = types.Int64Value(int64(mobileDevice.Hardware.UsedSpacePercentage))
	data.BatteryLevel = types.Int64Value(int64(mobileDevice.Hardware.BatteryLevel))
	data.BatteryHealth = types.StringValue(mobileDevice.Hardware.BatteryHealth)
	data.SerialNumber = types.StringValue(mobileDevice.Hardware.SerialNumber)
	data.HardwareWifiMacAddress = types.StringValue(mobileDevice.Hardware.WifiMacAddress)
	data.BluetoothMacAddress = types.StringValue(mobileDevice.Hardware.BluetoothMacAddress)
	data.Model = types.StringValue(mobileDevice.Hardware.Model)
	data.ModelIdentifier = types.StringValue(mobileDevice.Hardware.ModelIdentifier)
	data.ModelNumber = types.StringValue(mobileDevice.Hardware.ModelNumber)
	data.DeviceId = types.StringValue(mobileDevice.Hardware.DeviceId)
	data.Username = types.StringValue(mobileDevice.UserAndLocation.Username)
	data.RealName = types.StringValue(mobileDevice.UserAndLocation.RealName)
	data.EmailAddress = types.StringValue(mobileDevice.UserAndLocation.EmailAddress)
	data.Position = types.StringValue(mobileDevice.UserAndLocation.Position)
	data.PhoneNumber = types.StringValue(mobileDevice.UserAndLocation.PhoneNumber)
	data.DepartmentId = types.StringValue(mobileDevice.UserAndLocation.DepartmentId)
	data.BuildingId = types.StringValue(mobileDevice.UserAndLocation.BuildingId)
	data.Room = types.StringValue(mobileDevice.UserAndLocation.Room)
	data.Building = types.StringValue(mobileDevice.UserAndLocation.Building)
	data.Department = types.StringValue(mobileDevice.UserAndLocation.Department)
	data.Purchased = types.BoolValue(mobileDevice.Purchasing.Purchased)
	data.Leased = types.BoolValue(mobileDevice.Purchasing.Leased)
	data.PoNumber = types.StringValue(mobileDevice.Purchasing.PoNumber)
	data.Vendor = types.StringValue(mobileDevice.Purchasing.Vendor)
	data.AppleCareId = types.StringValue(mobileDevice.Purchasing.AppleCareId)
	data.PurchasePrice = types.StringValue(mobileDevice.Purchasing.PurchasePrice)
	data.PurchasingAccount = types.StringValue(mobileDevice.Purchasing.PurchasingAccount)
	data.PoDate = types.StringValue(mobileDevice.Purchasing.PoDate)
	data.WarrantyExpiresDate = types.StringValue(mobileDevice.Purchasing.WarrantyExpiresDate)
	data.LeaseExpiresDate = types.StringValue(mobileDevice.Purchasing.LeaseExpiresDate)
	data.LifeExpectancy = types.Int64Value(int64(mobileDevice.Purchasing.LifeExpectancy))
	data.PurchasingContact = types.StringValue(mobileDevice.Purchasing.PurchasingContact)
	data.DataProtected = types.BoolValue(mobileDevice.Security.DataProtected)
	data.BlockLevelEncryptionCapable = types.BoolValue(mobileDevice.Security.BlockLevelEncryptionCapable)
	data.FileLevelEncryptionCapable = types.BoolValue(mobileDevice.Security.FileLevelEncryptionCapable)
	data.PasscodePresent = types.BoolValue(mobileDevice.Security.PasscodePresent)
	data.PasscodeCompliant = types.BoolValue(mobileDevice.Security.PasscodeCompliant)
	data.ActivationLockEnabled = types.BoolValue(mobileDevice.Security.ActivationLockEnabled)
	data.JailBreakDetected = types.BoolValue(mobileDevice.Security.JailBreakDetected)
	data.LostModeEnabled = types.BoolValue(mobileDevice.Security.LostModeEnabled)
	data.LostModeMessage = types.StringValue(mobileDevice.Security.LostModeMessage)
	data.LostModePhoneNumber = types.StringValue(mobileDevice.Security.LostModePhoneNumber)
	data.CellularTechnology = types.StringValue(mobileDevice.Network.CellularTechnology)
	data.Iccid = types.StringValue(mobileDevice.Network.Iccid)
	data.Carrier = types.StringValue(mobileDevice.Network.Carrier)
	data.SimPhoneNumber = types.StringValue(mobileDevice.Network.SimPhoneNumber)
	data.NetworkWifiMacAddress = types.StringValue(mobileDevice.Network.WifiMacAddress)
	data.BluetoothMac = types.StringValue(mobileDevice.Network.BluetoothMac)
	data.EthernetMac = types.StringValue(mobileDevice.Network.EthernetMac)

	var appList []attr.Value
	for _, app := range mobileDevice.Applications {
		appAttrs := map[string]attr.Value{
			"identifier":        types.StringValue(app.Identifier),
			"name":              types.StringValue(app.Name),
			"version":           types.StringValue(app.Version),
			"short_version":     types.StringValue(app.ShortVersion),
			"management_status": types.StringValue(app.ManagementStatus),
			"bundle_size":       types.StringValue(app.BundleSize),
			"dynamic_size":      types.StringValue(app.DynamicSize),
		}
		appVal, diags := types.ObjectValue(map[string]attr.Type{
			"identifier":        types.StringType,
			"name":              types.StringType,
			"version":           types.StringType,
			"short_version":     types.StringType,
			"management_status": types.StringType,
			"bundle_size":       types.StringType,
			"dynamic_size":      types.StringType,
		}, appAttrs)
		resp.Diagnostics.Append(diags...)
		appList = append(appList, appVal)
	}

	applicationsVal, diags := types.ListValue(types.ObjectType{AttrTypes: map[string]attr.Type{
		"identifier":        types.StringType,
		"name":              types.StringType,
		"version":           types.StringType,
		"short_version":     types.StringType,
		"management_status": types.StringType,
		"bundle_size":       types.StringType,
		"dynamic_size":      types.StringType,
	}}, appList)
	resp.Diagnostics.Append(diags...)
	data.Applications = applicationsVal

	var profileList []attr.Value
	for _, profile := range mobileDevice.Profiles {
		profileAttrs := map[string]attr.Value{
			"display_name":   types.StringValue(profile.DisplayName),
			"version":        types.StringValue(profile.Version),
			"uuid":           types.StringValue(profile.Uuid),
			"identifier":     types.StringValue(profile.Identifier),
			"removable":      types.BoolValue(profile.Removable),
			"last_installed": types.StringValue(profile.LastInstalled),
			"username":       types.StringValue(profile.Username),
		}
		profileVal, diags := types.ObjectValue(map[string]attr.Type{
			"display_name":   types.StringType,
			"version":        types.StringType,
			"uuid":           types.StringType,
			"identifier":     types.StringType,
			"removable":      types.BoolType,
			"last_installed": types.StringType,
			"username":       types.StringType,
		}, profileAttrs)
		resp.Diagnostics.Append(diags...)
		profileList = append(profileList, profileVal)
	}

	profilesVal, diags := types.ListValue(types.ObjectType{AttrTypes: map[string]attr.Type{
		"display_name":   types.StringType,
		"version":        types.StringType,
		"uuid":           types.StringType,
		"identifier":     types.StringType,
		"removable":      types.BoolType,
		"last_installed": types.StringType,
		"username":       types.StringType,
	}}, profileList)
	resp.Diagnostics.Append(diags...)
	data.Profiles = profilesVal

	var certList []attr.Value
	for _, cert := range mobileDevice.Certificates {
		certAttrs := map[string]attr.Value{
			"common_name":     types.StringValue(cert.CommonName),
			"identity":        types.BoolValue(cert.Identity),
			"expiration_date": types.StringValue(cert.ExpirationDate),
		}
		certVal, diags := types.ObjectValue(map[string]attr.Type{
			"common_name":     types.StringType,
			"identity":        types.BoolType,
			"expiration_date": types.StringType,
		}, certAttrs)
		resp.Diagnostics.Append(diags...)
		certList = append(certList, certVal)
	}

	certificatesVal, diags := types.ListValue(types.ObjectType{AttrTypes: map[string]attr.Type{
		"common_name":     types.StringType,
		"identity":        types.BoolType,
		"expiration_date": types.StringType,
	}}, certList)
	resp.Diagnostics.Append(diags...)
	data.Certificates = certificatesVal

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "read a data source")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
