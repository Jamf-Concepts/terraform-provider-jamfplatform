// Copyright 2025 Jamf Software LLC.

package mobiledevice

import (
	"context"
	"fmt"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/client"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/shared"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// DataSourceMobileDevice defines the data source implementation.
type DataSourceMobileDevice struct {
	client *client.Client
}

// mobileDeviceDataSourceModel maps the data source schema data.
type mobileDeviceDataSourceModel struct {
	ID              types.String `tfsdk:"id"`
	MobileDeviceId  types.String `tfsdk:"mobile_device_id"`
	DeviceType      types.String `tfsdk:"device_type"`
	Sections        types.List   `tfsdk:"sections"`
	General         types.Object `tfsdk:"general"`
	Hardware        types.Object `tfsdk:"hardware"`
	UserAndLocation types.Object `tfsdk:"user_and_location"`
	Purchasing      types.Object `tfsdk:"purchasing"`
	Security        types.Object `tfsdk:"security"`
	Network         types.Object `tfsdk:"network"`
	Applications    types.List   `tfsdk:"applications"`
	Profiles        types.List   `tfsdk:"profiles"`
	Certificates    types.List   `tfsdk:"certificates"`
}

// Ensure DataSourceMobileDevice implements the datasource.DataSource interface.
var _ datasource.DataSource = &DataSourceMobileDevice{}

// NewDataSourceMobileDevice returns a new data source instance.
func NewDataSourceMobileDevice() datasource.DataSource {
	return &DataSourceMobileDevice{}
}

// Configure sets up the API client for the data source from the provider configuration.
func (d *DataSourceMobileDevice) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	providerClients, ok := req.ProviderData.(*shared.ProviderClients)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected ProviderData type",
			"Expected *shared.ProviderClients, got something else.",
		)
		return
	}
	if providerClients.Inventory == nil {
		resp.Diagnostics.AddError(
			"Inventory API client not configured",
			"The provider's inventory block is missing or incomplete. Please provide valid credentials.",
		)
		return
	}
	d.client = providerClients.Inventory
}

// Metadata sets the data source type name for the Terraform provider.
func (d *DataSourceMobileDevice) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_inventory_mobile_device"
}

// Schema defines the schema for the mobile device data source.
func (d *DataSourceMobileDevice) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
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
			"general": schema.SingleNestedAttribute{
				Computed:    true,
				Description: "General information about the mobile device.",
				Attributes: map[string]schema.Attribute{
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
				},
			},
			"hardware": schema.SingleNestedAttribute{
				Computed:    true,
				Description: "Hardware information.",
				Attributes: map[string]schema.Attribute{
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
					"wifi_mac_address": schema.StringAttribute{
						Computed:    true,
						Description: "WiFi MAC address.",
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
				},
			},
			"user_and_location": schema.SingleNestedAttribute{
				Computed:    true,
				Description: "User and location information.",
				Attributes: map[string]schema.Attribute{
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
				},
			},
			"purchasing": schema.SingleNestedAttribute{
				Computed:    true,
				Description: "Purchasing and warranty information.",
				Attributes: map[string]schema.Attribute{
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
				},
			},
			"security": schema.SingleNestedAttribute{
				Computed:    true,
				Description: "Security information.",
				Attributes: map[string]schema.Attribute{
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
				},
			},
			"network": schema.SingleNestedAttribute{
				Computed:    true,
				Description: "Network information.",
				Attributes: map[string]schema.Attribute{
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
					"wifi_mac_address": schema.StringAttribute{
						Computed:    true,
						Description: "WiFi MAC address.",
					},
					"bluetooth_mac": schema.StringAttribute{
						Computed:    true,
						Description: "Bluetooth MAC address.",
					},
					"ethernet_mac": schema.StringAttribute{
						Computed:    true,
						Description: "Ethernet MAC address.",
					},
				},
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

// Read fetches the mobile device details and sets the state.
func (d *DataSourceMobileDevice) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data mobileDeviceDataSourceModel
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

	mobileDevice, err := d.client.GetInventoryMobileDeviceByID(ctx, id, sections)
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

	generalAttrs := map[string]attr.Value{
		"udid":                       types.StringValue(mobileDevice.General.Udid),
		"display_name":               types.StringValue(mobileDevice.General.DisplayName),
		"asset_tag":                  types.StringValue(mobileDevice.General.AssetTag),
		"site_id":                    types.StringValue(mobileDevice.General.SiteId),
		"last_inventory_update_date": types.StringValue(mobileDevice.General.LastInventoryUpdateDate),
		"os_version":                 types.StringValue(mobileDevice.General.OsVersion),
		"os_build":                   types.StringValue(mobileDevice.General.OsBuild),
		"ip_address":                 types.StringValue(mobileDevice.General.IpAddress),
		"managed":                    types.BoolValue(mobileDevice.General.Managed),
		"supervised":                 types.BoolValue(mobileDevice.General.Supervised),
		"device_ownership_type":      types.StringValue(mobileDevice.General.DeviceOwnershipType),
		"last_enrolled_date":         types.StringValue(mobileDevice.General.LastEnrolledDate),
		"mdm_profile_expiration":     types.StringValue(mobileDevice.General.MdmProfileExpiration),
		"time_zone":                  types.StringValue(mobileDevice.General.TimeZone),
	}

	generalVal, diags := types.ObjectValue(map[string]attr.Type{
		"udid":                       types.StringType,
		"display_name":               types.StringType,
		"asset_tag":                  types.StringType,
		"site_id":                    types.StringType,
		"last_inventory_update_date": types.StringType,
		"os_version":                 types.StringType,
		"os_build":                   types.StringType,
		"ip_address":                 types.StringType,
		"managed":                    types.BoolType,
		"supervised":                 types.BoolType,
		"device_ownership_type":      types.StringType,
		"last_enrolled_date":         types.StringType,
		"mdm_profile_expiration":     types.StringType,
		"time_zone":                  types.StringType,
	}, generalAttrs)
	resp.Diagnostics.Append(diags...)
	data.General = generalVal

	hardwareAttrs := map[string]attr.Value{
		"capacity_mb":           types.Int64Value(int64(mobileDevice.Hardware.CapacityMb)),
		"available_space_mb":    types.Int64Value(int64(mobileDevice.Hardware.AvailableSpaceMb)),
		"used_space_percentage": types.Int64Value(int64(mobileDevice.Hardware.UsedSpacePercentage)),
		"battery_level":         types.Int64Value(int64(mobileDevice.Hardware.BatteryLevel)),
		"battery_health":        types.StringValue(mobileDevice.Hardware.BatteryHealth),
		"serial_number":         types.StringValue(mobileDevice.Hardware.SerialNumber),
		"wifi_mac_address":      types.StringValue(mobileDevice.Hardware.WifiMacAddress),
		"bluetooth_mac_address": types.StringValue(mobileDevice.Hardware.BluetoothMacAddress),
		"model":                 types.StringValue(mobileDevice.Hardware.Model),
		"model_identifier":      types.StringValue(mobileDevice.Hardware.ModelIdentifier),
		"model_number":          types.StringValue(mobileDevice.Hardware.ModelNumber),
		"device_id":             types.StringValue(mobileDevice.Hardware.DeviceId),
	}

	hardwareVal, diags := types.ObjectValue(map[string]attr.Type{
		"capacity_mb":           types.Int64Type,
		"available_space_mb":    types.Int64Type,
		"used_space_percentage": types.Int64Type,
		"battery_level":         types.Int64Type,
		"battery_health":        types.StringType,
		"serial_number":         types.StringType,
		"wifi_mac_address":      types.StringType,
		"bluetooth_mac_address": types.StringType,
		"model":                 types.StringType,
		"model_identifier":      types.StringType,
		"model_number":          types.StringType,
		"device_id":             types.StringType,
	}, hardwareAttrs)
	resp.Diagnostics.Append(diags...)
	data.Hardware = hardwareVal

	userLocationAttrs := map[string]attr.Value{
		"username":      types.StringValue(mobileDevice.UserAndLocation.Username),
		"real_name":     types.StringValue(mobileDevice.UserAndLocation.RealName),
		"email_address": types.StringValue(mobileDevice.UserAndLocation.EmailAddress),
		"position":      types.StringValue(mobileDevice.UserAndLocation.Position),
		"phone_number":  types.StringValue(mobileDevice.UserAndLocation.PhoneNumber),
		"department_id": types.StringValue(mobileDevice.UserAndLocation.DepartmentId),
		"building_id":   types.StringValue(mobileDevice.UserAndLocation.BuildingId),
		"room":          types.StringValue(mobileDevice.UserAndLocation.Room),
		"building":      types.StringValue(mobileDevice.UserAndLocation.Building),
		"department":    types.StringValue(mobileDevice.UserAndLocation.Department),
	}

	userLocationVal, diags := types.ObjectValue(map[string]attr.Type{
		"username":      types.StringType,
		"real_name":     types.StringType,
		"email_address": types.StringType,
		"position":      types.StringType,
		"phone_number":  types.StringType,
		"department_id": types.StringType,
		"building_id":   types.StringType,
		"room":          types.StringType,
		"building":      types.StringType,
		"department":    types.StringType,
	}, userLocationAttrs)
	resp.Diagnostics.Append(diags...)
	data.UserAndLocation = userLocationVal

	purchasingAttrs := map[string]attr.Value{
		"purchased":             types.BoolValue(mobileDevice.Purchasing.Purchased),
		"leased":                types.BoolValue(mobileDevice.Purchasing.Leased),
		"po_number":             types.StringValue(mobileDevice.Purchasing.PoNumber),
		"vendor":                types.StringValue(mobileDevice.Purchasing.Vendor),
		"apple_care_id":         types.StringValue(mobileDevice.Purchasing.AppleCareId),
		"purchase_price":        types.StringValue(mobileDevice.Purchasing.PurchasePrice),
		"purchasing_account":    types.StringValue(mobileDevice.Purchasing.PurchasingAccount),
		"po_date":               types.StringValue(mobileDevice.Purchasing.PoDate),
		"warranty_expires_date": types.StringValue(mobileDevice.Purchasing.WarrantyExpiresDate),
		"lease_expires_date":    types.StringValue(mobileDevice.Purchasing.LeaseExpiresDate),
		"life_expectancy":       types.Int64Value(int64(mobileDevice.Purchasing.LifeExpectancy)),
		"purchasing_contact":    types.StringValue(mobileDevice.Purchasing.PurchasingContact),
	}

	purchasingVal, diags := types.ObjectValue(map[string]attr.Type{
		"purchased":             types.BoolType,
		"leased":                types.BoolType,
		"po_number":             types.StringType,
		"vendor":                types.StringType,
		"apple_care_id":         types.StringType,
		"purchase_price":        types.StringType,
		"purchasing_account":    types.StringType,
		"po_date":               types.StringType,
		"warranty_expires_date": types.StringType,
		"lease_expires_date":    types.StringType,
		"life_expectancy":       types.Int64Type,
		"purchasing_contact":    types.StringType,
	}, purchasingAttrs)
	resp.Diagnostics.Append(diags...)
	data.Purchasing = purchasingVal

	securityAttrs := map[string]attr.Value{
		"data_protected":                 types.BoolValue(mobileDevice.Security.DataProtected),
		"block_level_encryption_capable": types.BoolValue(mobileDevice.Security.BlockLevelEncryptionCapable),
		"file_level_encryption_capable":  types.BoolValue(mobileDevice.Security.FileLevelEncryptionCapable),
		"passcode_present":               types.BoolValue(mobileDevice.Security.PasscodePresent),
		"passcode_compliant":             types.BoolValue(mobileDevice.Security.PasscodeCompliant),
		"activation_lock_enabled":        types.BoolValue(mobileDevice.Security.ActivationLockEnabled),
		"jail_break_detected":            types.BoolValue(mobileDevice.Security.JailBreakDetected),
		"lost_mode_enabled":              types.BoolValue(mobileDevice.Security.LostModeEnabled),
		"lost_mode_message":              types.StringValue(mobileDevice.Security.LostModeMessage),
		"lost_mode_phone_number":         types.StringValue(mobileDevice.Security.LostModePhoneNumber),
	}

	securityVal, diags := types.ObjectValue(map[string]attr.Type{
		"data_protected":                 types.BoolType,
		"block_level_encryption_capable": types.BoolType,
		"file_level_encryption_capable":  types.BoolType,
		"passcode_present":               types.BoolType,
		"passcode_compliant":             types.BoolType,
		"activation_lock_enabled":        types.BoolType,
		"jail_break_detected":            types.BoolType,
		"lost_mode_enabled":              types.BoolType,
		"lost_mode_message":              types.StringType,
		"lost_mode_phone_number":         types.StringType,
	}, securityAttrs)
	resp.Diagnostics.Append(diags...)
	data.Security = securityVal

	networkAttrs := map[string]attr.Value{
		"cellular_technology": types.StringValue(mobileDevice.Network.CellularTechnology),
		"iccid":               types.StringValue(mobileDevice.Network.Iccid),
		"carrier":             types.StringValue(mobileDevice.Network.Carrier),
		"sim_phone_number":    types.StringValue(mobileDevice.Network.SimPhoneNumber),
		"wifi_mac_address":    types.StringValue(mobileDevice.Network.WifiMacAddress),
		"bluetooth_mac":       types.StringValue(mobileDevice.Network.BluetoothMac),
		"ethernet_mac":        types.StringValue(mobileDevice.Network.EthernetMac),
	}

	networkVal, diags := types.ObjectValue(map[string]attr.Type{
		"cellular_technology": types.StringType,
		"iccid":               types.StringType,
		"carrier":             types.StringType,
		"sim_phone_number":    types.StringType,
		"wifi_mac_address":    types.StringType,
		"bluetooth_mac":       types.StringType,
		"ethernet_mac":        types.StringType,
	}, networkAttrs)
	resp.Diagnostics.Append(diags...)
	data.Network = networkVal

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

	resp.State.Set(ctx, &data)
}
