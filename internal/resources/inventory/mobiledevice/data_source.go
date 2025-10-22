// Copyright 2025 Jamf Software LLC.

package mobiledevice

import (
	"context"
	"fmt"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/client"
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
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the mobile device to retrieve.",
			},
			"mobile_device_id": schema.StringAttribute{
				Computed:    true,
				Description: "Mobile device ID from the API response.",
			},
			"device_type": schema.StringAttribute{
				Computed:    true,
				Description: "Type of the device.",
			},
			"sections": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Description: "Sections to retrieve (e.g., ['GENERAL', 'HARDWARE', 'SECURITY']). If not specified, all sections are retrieved.",
			},
			"udid": schema.StringAttribute{
				Computed:    true,
				Description: "UDID of the device.",
			},
			"display_name": schema.StringAttribute{
				Computed:    true,
				Description: "Display name of the device.",
			},
			"asset_tag": schema.StringAttribute{
				Computed:    true,
				Description: "Asset tag.",
			},
			"site_id": schema.StringAttribute{
				Computed:    true,
				Description: "Site ID.",
			},
			"last_inventory_update_date": schema.StringAttribute{
				Computed:    true,
				Description: "Last inventory update date.",
			},
			"os_version": schema.StringAttribute{
				Computed:    true,
				Description: "OS version.",
			},
			"os_build": schema.StringAttribute{
				Computed:    true,
				Description: "OS build.",
			},
			"ip_address": schema.StringAttribute{
				Computed:    true,
				Description: "IP address.",
			},
			"managed": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the device is managed.",
			},
			"supervised": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the device is supervised.",
			},
			"device_ownership_type": schema.StringAttribute{
				Computed:    true,
				Description: "Device ownership type.",
			},
			"last_enrolled_date": schema.StringAttribute{
				Computed:    true,
				Description: "Last enrolled date.",
			},
			"mdm_profile_expiration": schema.StringAttribute{
				Computed:    true,
				Description: "MDM profile expiration date.",
			},
			"time_zone": schema.StringAttribute{
				Computed:    true,
				Description: "Device time zone.",
			},
			"capacity_mb": schema.Int64Attribute{
				Computed:    true,
				Description: "Storage capacity in MB.",
			},
			"available_space_mb": schema.Int64Attribute{
				Computed:    true,
				Description: "Available space in MB.",
			},
			"used_space_percentage": schema.Int64Attribute{
				Computed:    true,
				Description: "Used space percentage.",
			},
			"battery_level": schema.Int64Attribute{
				Computed:    true,
				Description: "Battery level percentage.",
			},
			"battery_health": schema.StringAttribute{
				Computed:    true,
				Description: "Battery health status.",
			},
			"serial_number": schema.StringAttribute{
				Computed:    true,
				Description: "Serial number.",
			},
			"hardware_wifi_mac_address": schema.StringAttribute{
				Computed:    true,
				Description: "WiFi MAC address from hardware section.",
			},
			"bluetooth_mac_address": schema.StringAttribute{
				Computed:    true,
				Description: "Bluetooth MAC address.",
			},
			"model": schema.StringAttribute{
				Computed:    true,
				Description: "Device model.",
			},
			"model_identifier": schema.StringAttribute{
				Computed:    true,
				Description: "Model identifier.",
			},
			"model_number": schema.StringAttribute{
				Computed:    true,
				Description: "Model number.",
			},
			"device_id": schema.StringAttribute{
				Computed:    true,
				Description: "Device ID.",
			},
			"username": schema.StringAttribute{
				Computed:    true,
				Description: "Username.",
			},
			"real_name": schema.StringAttribute{
				Computed:    true,
				Description: "Real name.",
			},
			"email_address": schema.StringAttribute{
				Computed:    true,
				Description: "Email address.",
			},
			"position": schema.StringAttribute{
				Computed:    true,
				Description: "Position.",
			},
			"phone_number": schema.StringAttribute{
				Computed:    true,
				Description: "Phone number.",
			},
			"department_id": schema.StringAttribute{
				Computed:    true,
				Description: "Department ID.",
			},
			"building_id": schema.StringAttribute{
				Computed:    true,
				Description: "Building ID.",
			},
			"room": schema.StringAttribute{
				Computed:    true,
				Description: "Room.",
			},
			"building": schema.StringAttribute{
				Computed:    true,
				Description: "Building name.",
			},
			"department": schema.StringAttribute{
				Computed:    true,
				Description: "Department name.",
			},
			"purchased": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether purchased.",
			},
			"leased": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether leased.",
			},
			"po_number": schema.StringAttribute{
				Computed:    true,
				Description: "Purchase order number.",
			},
			"vendor": schema.StringAttribute{
				Computed:    true,
				Description: "Vendor.",
			},
			"apple_care_id": schema.StringAttribute{
				Computed:    true,
				Description: "AppleCare ID.",
			},
			"purchase_price": schema.StringAttribute{
				Computed:    true,
				Description: "Purchase price.",
			},
			"purchasing_account": schema.StringAttribute{
				Computed:    true,
				Description: "Purchasing account.",
			},
			"po_date": schema.StringAttribute{
				Computed:    true,
				Description: "Purchase order date.",
			},
			"warranty_expires_date": schema.StringAttribute{
				Computed:    true,
				Description: "Warranty expiration date.",
			},
			"lease_expires_date": schema.StringAttribute{
				Computed:    true,
				Description: "Lease expiration date.",
			},
			"life_expectancy": schema.Int64Attribute{
				Computed:    true,
				Description: "Life expectancy in years.",
			},
			"purchasing_contact": schema.StringAttribute{
				Computed:    true,
				Description: "Purchasing contact.",
			},
			"data_protected": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether data is protected.",
			},
			"block_level_encryption_capable": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether block level encryption capable.",
			},
			"file_level_encryption_capable": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether file level encryption capable.",
			},
			"passcode_present": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether passcode is present.",
			},
			"passcode_compliant": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether passcode is compliant.",
			},
			"activation_lock_enabled": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether activation lock is enabled.",
			},
			"jail_break_detected": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether jailbreak is detected.",
			},
			"lost_mode_enabled": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether lost mode is enabled.",
			},
			"lost_mode_message": schema.StringAttribute{
				Computed:    true,
				Description: "Lost mode message.",
			},
			"lost_mode_phone_number": schema.StringAttribute{
				Computed:    true,
				Description: "Lost mode phone number.",
			},
			"cellular_technology": schema.StringAttribute{
				Computed:    true,
				Description: "Cellular technology.",
			},
			"iccid": schema.StringAttribute{
				Computed:    true,
				Description: "ICCID.",
			},
			"carrier": schema.StringAttribute{
				Computed:    true,
				Description: "Carrier.",
			},
			"sim_phone_number": schema.StringAttribute{
				Computed:    true,
				Description: "SIM phone number.",
			},
			"network_wifi_mac_address": schema.StringAttribute{
				Computed:    true,
				Description: "WiFi MAC address from network section.",
			},
			"bluetooth_mac": schema.StringAttribute{
				Computed:    true,
				Description: "Bluetooth MAC address.",
			},
			"ethernet_mac": schema.StringAttribute{
				Computed:    true,
				Description: "Ethernet MAC address.",
			},
			"applications": schema.ListNestedAttribute{
				Computed:    true,
				Description: "Applications installed on the device.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"identifier": schema.StringAttribute{
							Computed:    true,
							Description: "Application identifier.",
						},
						"name": schema.StringAttribute{
							Computed:    true,
							Description: "Application name.",
						},
						"version": schema.StringAttribute{
							Computed:    true,
							Description: "Application version.",
						},
						"short_version": schema.StringAttribute{
							Computed:    true,
							Description: "Short version.",
						},
						"management_status": schema.StringAttribute{
							Computed:    true,
							Description: "Management status.",
						},
						"bundle_size": schema.StringAttribute{
							Computed:    true,
							Description: "Bundle size.",
						},
						"dynamic_size": schema.StringAttribute{
							Computed:    true,
							Description: "Dynamic size.",
						},
					},
				},
			},
			"profiles": schema.ListNestedAttribute{
				Computed:    true,
				Description: "Configuration profiles installed on the device.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"display_name": schema.StringAttribute{
							Computed:    true,
							Description: "Display name.",
						},
						"version": schema.StringAttribute{
							Computed:    true,
							Description: "Profile version.",
						},
						"uuid": schema.StringAttribute{
							Computed:    true,
							Description: "Profile UUID.",
						},
						"identifier": schema.StringAttribute{
							Computed:    true,
							Description: "Profile identifier.",
						},
						"removable": schema.BoolAttribute{
							Computed:    true,
							Description: "Whether removable.",
						},
						"last_installed": schema.StringAttribute{
							Computed:    true,
							Description: "Last installed date.",
						},
						"username": schema.StringAttribute{
							Computed:    true,
							Description: "Username.",
						},
					},
				},
			},
			"certificates": schema.ListNestedAttribute{
				Computed:    true,
				Description: "Certificates installed on the device.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"common_name": schema.StringAttribute{
							Computed:    true,
							Description: "Common name.",
						},
						"identity": schema.BoolAttribute{
							Computed:    true,
							Description: "Whether identity certificate.",
						},
						"expiration_date": schema.StringAttribute{
							Computed:    true,
							Description: "Expiration date.",
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
	if !data.Sections.IsNull() && !data.Sections.IsUnknown() {
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
