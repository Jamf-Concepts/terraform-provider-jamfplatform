// Copyright 2025 Jamf Software LLC.

package computer

import (
	"context"
	"fmt"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// DataSourceComputer defines the data source implementation.
type DataSourceComputer struct {
	client *client.Client
}

// computerDataSourceModel maps the data source schema data.
type computerDataSourceModel struct {
	ID                    types.String `tfsdk:"id"`
	UDID                  types.String `tfsdk:"udid"`
	General               types.Object `tfsdk:"general"`
	Hardware              types.Object `tfsdk:"hardware"`
	OperatingSystem       types.Object `tfsdk:"operating_system"`
	UserAndLocation       types.Object `tfsdk:"user_and_location"`
	Purchasing            types.Object `tfsdk:"purchasing"`
	Security              types.Object `tfsdk:"security"`
	Applications          types.List   `tfsdk:"applications"`
	ConfigurationProfiles types.List   `tfsdk:"configuration_profiles"`
	LocalUserAccounts     types.List   `tfsdk:"local_user_accounts"`
}

// Ensure DataSourceComputer implements the datasource.DataSource interface.
var _ datasource.DataSource = &DataSourceComputer{}

// NewDataSourceComputer returns a new data source instance.
func NewDataSourceComputer() datasource.DataSource {
	return &DataSourceComputer{}
}

// Configure sets up the API client for the data source from the provider configuration.
func (d *DataSourceComputer) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	apiClient, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected ProviderData type",
			"Expected *client.Client, got something else.",
		)
		return
	}
	d.client = apiClient
}

// Metadata sets the data source type name for the Terraform provider.
func (d *DataSourceComputer) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_inventory_computer"
}

// Schema sets the Terraform schema for the data source.
func (d *DataSourceComputer) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the computer to retrieve.",
			},
			"udid": schema.StringAttribute{
				Computed:    true,
				Description: "The UDID of the computer.",
			},
			"general": schema.SingleNestedAttribute{
				Computed:    true,
				Description: "General information about the computer.",
				Attributes: map[string]schema.Attribute{
					"name": schema.StringAttribute{
						Computed:    true,
						Description: "Name of the computer.",
					},
					"last_ip_address": schema.StringAttribute{
						Computed:    true,
						Description: "Last known IP address.",
					},
					"last_contact_time": schema.StringAttribute{
						Computed:    true,
						Description: "Last contact time.",
					},
					"last_enrolled_date": schema.StringAttribute{
						Computed:    true,
						Description: "Last enrolled date.",
					},
					"platform": schema.StringAttribute{
						Computed:    true,
						Description: "Platform of the computer.",
					},
					"supervised": schema.BoolAttribute{
						Computed:    true,
						Description: "Whether the computer is supervised.",
					},
					"asset_tag": schema.StringAttribute{
						Computed:    true,
						Description: "Asset tag.",
					},
					"jamf_binary_version": schema.StringAttribute{
						Computed:    true,
						Description: "Jamf binary version.",
					},
					"management_id": schema.StringAttribute{
						Computed:    true,
						Description: "Management ID.",
					},
				},
			},
			"hardware": schema.SingleNestedAttribute{
				Computed:    true,
				Description: "Hardware information.",
				Attributes: map[string]schema.Attribute{
					"make": schema.StringAttribute{
						Computed:    true,
						Description: "Hardware make.",
					},
					"model": schema.StringAttribute{
						Computed:    true,
						Description: "Hardware model.",
					},
					"model_identifier": schema.StringAttribute{
						Computed:    true,
						Description: "Model identifier.",
					},
					"serial_number": schema.StringAttribute{
						Computed:    true,
						Description: "Serial number.",
					},
					"processor_type": schema.StringAttribute{
						Computed:    true,
						Description: "Processor type.",
					},
					"processor_speed_mhz": schema.Int64Attribute{
						Computed:    true,
						Description: "Processor speed in MHz.",
					},
					"total_ram_megabytes": schema.Int64Attribute{
						Computed:    true,
						Description: "Total RAM in megabytes.",
					},
					"mac_address": schema.StringAttribute{
						Computed:    true,
						Description: "MAC address.",
					},
				},
			},
			"operating_system": schema.SingleNestedAttribute{
				Computed:    true,
				Description: "Operating system information.",
				Attributes: map[string]schema.Attribute{
					"name": schema.StringAttribute{
						Computed:    true,
						Description: "OS name.",
					},
					"version": schema.StringAttribute{
						Computed:    true,
						Description: "OS version.",
					},
					"build": schema.StringAttribute{
						Computed:    true,
						Description: "OS build.",
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
					"realname": schema.StringAttribute{
						Computed:    true,
						Description: "Real name.",
					},
					"email": schema.StringAttribute{
						Computed:    true,
						Description: "Email address.",
					},
					"position": schema.StringAttribute{
						Computed:    true,
						Description: "Position.",
					},
					"phone": schema.StringAttribute{
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
				},
			},
			"purchasing": schema.SingleNestedAttribute{
				Computed:    true,
				Description: "Purchasing and warranty information.",
				Attributes: map[string]schema.Attribute{
					"purchased": schema.BoolAttribute{
						Computed:    true,
						Description: "Whether the computer is purchased.",
					},
					"leased": schema.BoolAttribute{
						Computed:    true,
						Description: "Whether the computer is leased.",
					},
					"po_number": schema.StringAttribute{
						Computed:    true,
						Description: "Purchase order number.",
					},
					"vendor": schema.StringAttribute{
						Computed:    true,
						Description: "Vendor.",
					},
					"warranty_date": schema.StringAttribute{
						Computed:    true,
						Description: "Warranty date.",
					},
					"purchase_price": schema.StringAttribute{
						Computed:    true,
						Description: "Purchase price.",
					},
				},
			},
			"security": schema.SingleNestedAttribute{
				Computed:    true,
				Description: "Security information.",
				Attributes: map[string]schema.Attribute{
					"sip_status": schema.StringAttribute{
						Computed:    true,
						Description: "SIP status.",
					},
					"gatekeeper_status": schema.StringAttribute{
						Computed:    true,
						Description: "Gatekeeper status.",
					},
					"activation_lock_enabled": schema.BoolAttribute{
						Computed:    true,
						Description: "Whether activation lock is enabled.",
					},
					"recovery_lock_enabled": schema.BoolAttribute{
						Computed:    true,
						Description: "Whether recovery lock is enabled.",
					},
				},
			},
			"applications": schema.ListNestedAttribute{
				Computed:    true,
				Description: "Applications installed on the computer.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Computed:    true,
							Description: "Application name.",
						},
						"version": schema.StringAttribute{
							Computed:    true,
							Description: "Application version.",
						},
						"bundle_id": schema.StringAttribute{
							Computed:    true,
							Description: "Bundle ID.",
						},
						"size_megabytes": schema.Int64Attribute{
							Computed:    true,
							Description: "Size in megabytes.",
						},
						"mac_app_store": schema.BoolAttribute{
							Computed:    true,
							Description: "Whether from Mac App Store.",
						},
					},
				},
			},
			"configuration_profiles": schema.ListNestedAttribute{
				Computed:    true,
				Description: "Configuration profiles installed on the computer.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:    true,
							Description: "Profile ID.",
						},
						"display_name": schema.StringAttribute{
							Computed:    true,
							Description: "Display name.",
						},
						"profile_identifier": schema.StringAttribute{
							Computed:    true,
							Description: "Profile identifier.",
						},
						"removable": schema.BoolAttribute{
							Computed:    true,
							Description: "Whether removable.",
						},
					},
				},
			},
			"local_user_accounts": schema.ListNestedAttribute{
				Computed:    true,
				Description: "Local user accounts on the computer.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"username": schema.StringAttribute{
							Computed:    true,
							Description: "Username.",
						},
						"full_name": schema.StringAttribute{
							Computed:    true,
							Description: "Full name.",
						},
						"admin": schema.BoolAttribute{
							Computed:    true,
							Description: "Whether admin user.",
						},
						"home_directory": schema.StringAttribute{
							Computed:    true,
							Description: "Home directory.",
						},
					},
				},
			},
		},
	}
}

// Read retrieves the computer details based on the provided ID.
func (d *DataSourceComputer) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data computerDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := data.ID.ValueString()
	if id == "" {
		resp.Diagnostics.AddError(
			"Missing computer ID",
			"Computer ID is required to retrieve computer details.",
		)
		return
	}

	computer, err := d.client.GetInventoryComputerByID(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to get computer",
			fmt.Sprintf("Error retrieving computer with ID %s: %s", id, err),
		)
		return
	}

	data.ID = types.StringValue(computer.ID)
	data.UDID = types.StringValue(computer.UDID)

	generalAttrs := map[string]attr.Value{
		"name":                types.StringValue(computer.General.Name),
		"last_ip_address":     types.StringValue(computer.General.LastIpAddress),
		"last_contact_time":   types.StringValue(computer.General.LastContactTime),
		"last_enrolled_date":  types.StringValue(computer.General.LastEnrolledDate),
		"platform":            types.StringValue(computer.General.Platform),
		"supervised":          types.BoolValue(computer.General.Supervised),
		"asset_tag":           types.StringValue(computer.General.AssetTag),
		"jamf_binary_version": types.StringValue(computer.General.JamfBinaryVersion),
		"management_id":       types.StringValue(computer.General.ManagementId),
	}

	generalVal, diags := types.ObjectValue(map[string]attr.Type{
		"name":                types.StringType,
		"last_ip_address":     types.StringType,
		"last_contact_time":   types.StringType,
		"last_enrolled_date":  types.StringType,
		"platform":            types.StringType,
		"supervised":          types.BoolType,
		"asset_tag":           types.StringType,
		"jamf_binary_version": types.StringType,
		"management_id":       types.StringType,
	}, generalAttrs)
	resp.Diagnostics.Append(diags...)
	data.General = generalVal

	hardwareAttrs := map[string]attr.Value{
		"make":                types.StringValue(computer.Hardware.Make),
		"model":               types.StringValue(computer.Hardware.Model),
		"model_identifier":    types.StringValue(computer.Hardware.ModelIdentifier),
		"serial_number":       types.StringValue(computer.Hardware.SerialNumber),
		"processor_type":      types.StringValue(computer.Hardware.ProcessorType),
		"processor_speed_mhz": types.Int64Value(int64(computer.Hardware.ProcessorSpeedMhz)),
		"total_ram_megabytes": types.Int64Value(int64(computer.Hardware.TotalRamMegabytes)),
		"mac_address":         types.StringValue(computer.Hardware.MacAddress),
	}

	hardwareVal, diags := types.ObjectValue(map[string]attr.Type{
		"make":                types.StringType,
		"model":               types.StringType,
		"model_identifier":    types.StringType,
		"serial_number":       types.StringType,
		"processor_type":      types.StringType,
		"processor_speed_mhz": types.Int64Type,
		"total_ram_megabytes": types.Int64Type,
		"mac_address":         types.StringType,
	}, hardwareAttrs)
	resp.Diagnostics.Append(diags...)
	data.Hardware = hardwareVal

	osAttrs := map[string]attr.Value{
		"name":    types.StringValue(computer.OperatingSystem.Name),
		"version": types.StringValue(computer.OperatingSystem.Version),
		"build":   types.StringValue(computer.OperatingSystem.Build),
	}

	osVal, diags := types.ObjectValue(map[string]attr.Type{
		"name":    types.StringType,
		"version": types.StringType,
		"build":   types.StringType,
	}, osAttrs)
	resp.Diagnostics.Append(diags...)
	data.OperatingSystem = osVal

	userLocationAttrs := map[string]attr.Value{
		"username":      types.StringValue(computer.UserAndLocation.Username),
		"realname":      types.StringValue(computer.UserAndLocation.Realname),
		"email":         types.StringValue(computer.UserAndLocation.Email),
		"position":      types.StringValue(computer.UserAndLocation.Position),
		"phone":         types.StringValue(computer.UserAndLocation.Phone),
		"department_id": types.StringValue(computer.UserAndLocation.DepartmentId),
		"building_id":   types.StringValue(computer.UserAndLocation.BuildingId),
		"room":          types.StringValue(computer.UserAndLocation.Room),
	}

	userLocationVal, diags := types.ObjectValue(map[string]attr.Type{
		"username":      types.StringType,
		"realname":      types.StringType,
		"email":         types.StringType,
		"position":      types.StringType,
		"phone":         types.StringType,
		"department_id": types.StringType,
		"building_id":   types.StringType,
		"room":          types.StringType,
	}, userLocationAttrs)
	resp.Diagnostics.Append(diags...)
	data.UserAndLocation = userLocationVal

	purchasingAttrs := map[string]attr.Value{
		"purchased":      types.BoolValue(computer.Purchasing.Purchased),
		"leased":         types.BoolValue(computer.Purchasing.Leased),
		"po_number":      types.StringValue(computer.Purchasing.PoNumber),
		"vendor":         types.StringValue(computer.Purchasing.Vendor),
		"warranty_date":  types.StringValue(computer.Purchasing.WarrantyDate),
		"purchase_price": types.StringValue(computer.Purchasing.PurchasePrice),
	}

	purchasingVal, diags := types.ObjectValue(map[string]attr.Type{
		"purchased":      types.BoolType,
		"leased":         types.BoolType,
		"po_number":      types.StringType,
		"vendor":         types.StringType,
		"warranty_date":  types.StringType,
		"purchase_price": types.StringType,
	}, purchasingAttrs)
	resp.Diagnostics.Append(diags...)
	data.Purchasing = purchasingVal

	securityAttrs := map[string]attr.Value{
		"sip_status":              types.StringValue(computer.Security.SipStatus),
		"gatekeeper_status":       types.StringValue(computer.Security.GatekeeperStatus),
		"activation_lock_enabled": types.BoolValue(computer.Security.ActivationLockEnabled),
		"recovery_lock_enabled":   types.BoolValue(computer.Security.RecoveryLockEnabled),
	}

	securityVal, diags := types.ObjectValue(map[string]attr.Type{
		"sip_status":              types.StringType,
		"gatekeeper_status":       types.StringType,
		"activation_lock_enabled": types.BoolType,
		"recovery_lock_enabled":   types.BoolType,
	}, securityAttrs)
	resp.Diagnostics.Append(diags...)
	data.Security = securityVal

	var appList []attr.Value
	for _, app := range computer.Applications {
		appAttrs := map[string]attr.Value{
			"name":           types.StringValue(app.Name),
			"version":        types.StringValue(app.Version),
			"bundle_id":      types.StringValue(app.BundleId),
			"size_megabytes": types.Int64Value(int64(app.SizeMegabytes)),
			"mac_app_store":  types.BoolValue(app.MacAppStore),
		}
		appVal, diags := types.ObjectValue(map[string]attr.Type{
			"name":           types.StringType,
			"version":        types.StringType,
			"bundle_id":      types.StringType,
			"size_megabytes": types.Int64Type,
			"mac_app_store":  types.BoolType,
		}, appAttrs)
		resp.Diagnostics.Append(diags...)
		appList = append(appList, appVal)
	}

	applicationsVal, diags := types.ListValue(types.ObjectType{AttrTypes: map[string]attr.Type{
		"name":           types.StringType,
		"version":        types.StringType,
		"bundle_id":      types.StringType,
		"size_megabytes": types.Int64Type,
		"mac_app_store":  types.BoolType,
	}}, appList)
	resp.Diagnostics.Append(diags...)
	data.Applications = applicationsVal

	var profileList []attr.Value
	for _, profile := range computer.ConfigurationProfiles {
		profileAttrs := map[string]attr.Value{
			"id":                 types.StringValue(profile.ID),
			"display_name":       types.StringValue(profile.DisplayName),
			"profile_identifier": types.StringValue(profile.ProfileIdentifier),
			"removable":          types.BoolValue(profile.Removable),
		}
		profileVal, diags := types.ObjectValue(map[string]attr.Type{
			"id":                 types.StringType,
			"display_name":       types.StringType,
			"profile_identifier": types.StringType,
			"removable":          types.BoolType,
		}, profileAttrs)
		resp.Diagnostics.Append(diags...)
		profileList = append(profileList, profileVal)
	}

	configProfilesVal, diags := types.ListValue(types.ObjectType{AttrTypes: map[string]attr.Type{
		"id":                 types.StringType,
		"display_name":       types.StringType,
		"profile_identifier": types.StringType,
		"removable":          types.BoolType,
	}}, profileList)
	resp.Diagnostics.Append(diags...)
	data.ConfigurationProfiles = configProfilesVal

	var userList []attr.Value
	for _, user := range computer.LocalUserAccounts {
		userAttrs := map[string]attr.Value{
			"username":       types.StringValue(user.Username),
			"full_name":      types.StringValue(user.FullName),
			"admin":          types.BoolValue(user.Admin),
			"home_directory": types.StringValue(user.HomeDirectory),
		}
		userVal, diags := types.ObjectValue(map[string]attr.Type{
			"username":       types.StringType,
			"full_name":      types.StringType,
			"admin":          types.BoolType,
			"home_directory": types.StringType,
		}, userAttrs)
		resp.Diagnostics.Append(diags...)
		userList = append(userList, userVal)
	}

	localUsersVal, diags := types.ListValue(types.ObjectType{AttrTypes: map[string]attr.Type{
		"username":       types.StringType,
		"full_name":      types.StringType,
		"admin":          types.BoolType,
		"home_directory": types.StringType,
	}}, userList)
	resp.Diagnostics.Append(diags...)
	data.LocalUserAccounts = localUsersVal

	if resp.Diagnostics.HasError() {
		return
	}

	resp.State.Set(ctx, &data)
}
