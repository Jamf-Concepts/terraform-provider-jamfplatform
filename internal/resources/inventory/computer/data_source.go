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
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &DataSourceComputer{}

// NewDataSourceComputer returns a new instance of DataSourceComputer.
func NewDataSourceComputer() datasource.DataSource {
	return &DataSourceComputer{}
}

// Metadata sets the data source type name for the Terraform provider.
func (d *DataSourceComputer) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_inventory_computer"
}

// Schema sets the Terraform schema for the data source.
func (d *DataSourceComputer) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lookup a Jamf computer by ID from an Environment via the Universal Inventory API. Requires **Inventory API** access.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The ID of the computer to retrieve.",
			},
			"udid": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The UDID of the computer.",
			},
			"name": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Name of the computer.",
			},
			"last_ip_address": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Last known IP address.",
			},
			"last_contact_time": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Last contact time.",
			},
			"last_enrolled_date": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Last enrolled date.",
			},
			"platform": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Platform of the computer.",
			},
			"supervised": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the computer is supervised.",
			},
			"asset_tag": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Asset tag.",
			},
			"jamf_binary_version": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Jamf binary version.",
			},
			"management_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Management ID.",
			},
			"make": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Hardware make.",
			},
			"model": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Hardware model.",
			},
			"model_identifier": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Model identifier.",
			},
			"serial_number": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Serial number.",
			},
			"processor_type": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Processor type.",
			},
			"processor_speed_mhz": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Processor speed in MHz.",
			},
			"total_ram_megabytes": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Total RAM in megabytes.",
			},
			"mac_address": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "MAC address.",
			},
			"os_name": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "OS name.",
			},
			"os_version": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "OS version.",
			},
			"os_build": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "OS build.",
			},
			"username": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Username.",
			},
			"realname": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Real name.",
			},
			"email": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Email address.",
			},
			"position": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Position.",
			},
			"phone": schema.StringAttribute{
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
			"purchased": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the computer is purchased.",
			},
			"leased": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the computer is leased.",
			},
			"po_number": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Purchase order number.",
			},
			"vendor": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Vendor.",
			},
			"warranty_date": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Warranty date.",
			},
			"purchase_price": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Purchase price.",
			},
			"sip_status": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "SIP status.",
			},
			"gatekeeper_status": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Gatekeeper status.",
			},
			"activation_lock_enabled": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether activation lock is enabled.",
			},
			"recovery_lock_enabled": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether recovery lock is enabled.",
			},
			"applications": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Applications installed on the computer.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name":           schema.StringAttribute{Computed: true, MarkdownDescription: "Application name."},
						"version":        schema.StringAttribute{Computed: true, MarkdownDescription: "Application version."},
						"bundle_id":      schema.StringAttribute{Computed: true, MarkdownDescription: "Bundle ID."},
						"size_megabytes": schema.Int64Attribute{Computed: true, MarkdownDescription: "Size in megabytes."},
						"mac_app_store":  schema.BoolAttribute{Computed: true, MarkdownDescription: "Whether from Mac App Store."},
					},
				},
			},
			"configuration_profiles": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Configuration profiles installed on the computer.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":                 schema.StringAttribute{Computed: true, MarkdownDescription: "Profile ID."},
						"display_name":       schema.StringAttribute{Computed: true, MarkdownDescription: "Display name."},
						"profile_identifier": schema.StringAttribute{Computed: true, MarkdownDescription: "Profile identifier."},
						"removable":          schema.BoolAttribute{Computed: true, MarkdownDescription: "Whether removable."},
					},
				},
			},
			"local_user_accounts": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Local user accounts on the computer.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"username":       schema.StringAttribute{Computed: true, MarkdownDescription: "Username."},
						"full_name":      schema.StringAttribute{Computed: true, MarkdownDescription: "Full name."},
						"admin":          schema.BoolAttribute{Computed: true, MarkdownDescription: "Whether admin user."},
						"home_directory": schema.StringAttribute{Computed: true, MarkdownDescription: "Home directory."},
					},
				},
			},
		},
	}
}

// Configure sets up the API client for the data source from the provider configuration.
func (d *DataSourceComputer) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

// Read fetches the data for the data source.
func (d *DataSourceComputer) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ComputerDataSourceModel

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

	computer, err := d.client.GetInventoryComputerByIDV1(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to get computer",
			fmt.Sprintf("Error retrieving computer with ID %s: %s", id, err),
		)
		return
	}

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
	configurationProfilesVal, diags := types.ListValue(types.ObjectType{AttrTypes: map[string]attr.Type{
		"id":                 types.StringType,
		"display_name":       types.StringType,
		"profile_identifier": types.StringType,
		"removable":          types.BoolType,
	}}, profileList)
	resp.Diagnostics.Append(diags...)
	data.ConfigurationProfiles = configurationProfilesVal

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
	localUserAccountsVal, diags := types.ListValue(types.ObjectType{AttrTypes: map[string]attr.Type{
		"username":       types.StringType,
		"full_name":      types.StringType,
		"admin":          types.BoolType,
		"home_directory": types.StringType,
	}}, userList)
	resp.Diagnostics.Append(diags...)
	data.LocalUserAccounts = localUserAccountsVal

	tflog.Trace(ctx, "read a data source")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
