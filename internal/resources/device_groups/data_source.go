// Copyright 2025 Jamf Software LLC.

package device_groups

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/datasource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const defaultDeviceGroupsReadTimeout = 90 * time.Second

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &DeviceGroupsDataSource{}

// NewDeviceGroupsDataSource returns a new instance of DeviceGroupsDataSource.
func NewDeviceGroupsDataSource() datasource.DataSource {
	return &DeviceGroupsDataSource{}
}

// Metadata sets the data source type name for the Terraform provider.
func (d *DeviceGroupsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_device_groups"
}

// Schema sets the Terraform schema for the data source.
func (d *DeviceGroupsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Search Jamf device groups using optional filters. Requires **Device Group Inventory API** access.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Internal identifier for this data source read.",
				Computed:            true,
			},
		},
		Blocks: map[string]schema.Block{
			"filter": schema.ListNestedBlock{
				MarkdownDescription: "Declarative RSQL filter clauses. Each block represents one selector/operator/argument clause.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"selector": schema.StringAttribute{
							MarkdownDescription: "RSQL selector for device groups. Valid values are `name`, `description`, `deviceType`, and `groupType`.",
							Required:            true,
							Validators: []validator.String{
								stringvalidator.OneOf("name", "description", "deviceType", "groupType"),
							},
						},
						"operator": schema.StringAttribute{
							MarkdownDescription: "RSQL comparison operator such as `==`, `!=`, `=in=`, `>`, etc. Defaults to `==` when omitted.",
							Optional:            true,
						},
						"argument": schema.StringAttribute{
							MarkdownDescription: "RSQL argument portion for the selector/operator. Provide the value exactly as required by the API. The provider automatically escapes embedded double quotes and wraps the argument in double quotes whenever it contains RSQL-reserved characters (for example commas or spaces). Supply your own quoting only when you need custom list expressions such as those used with `=in=`.",
							Required:            true,
						},
						"join_with": schema.StringAttribute{
							MarkdownDescription: "Logical operator used to join this clause with the previous one. Valid values are `and` and `or`. Defaults to `and` when omitted or for the first block.",
							Optional:            true,
							Validators: []validator.String{
								stringvalidator.OneOf("and", "or"),
							},
						},
					},
				},
			},
			"device_groups": schema.ListNestedBlock{
				MarkdownDescription: "Device groups that matched the applied filters.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "Device group Platform ID.",
							Computed:            true,
						},
						"name": schema.StringAttribute{
							MarkdownDescription: "Device group name.",
							Computed:            true,
						},
						"description": schema.StringAttribute{
							MarkdownDescription: "Device group description.",
							Computed:            true,
						},
						"device_type": schema.StringAttribute{
							MarkdownDescription: "Device type for the group (lowercase).",
							Computed:            true,
						},
						"group_type": schema.StringAttribute{
							MarkdownDescription: "Group type for the group (lowercase).",
							Computed:            true,
						},
						"member_count": schema.Int64Attribute{
							MarkdownDescription: "Number of members in the group.",
							Computed:            true,
						},
					},
				},
			},
			"timeouts": timeouts.Block(ctx),
		},
	}
}

// Configure sets up the API client for the data source from the provider configuration.
func (d *DeviceGroupsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

// Read fetches device groups that match the provided filters.
func (d *DeviceGroupsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data DeviceGroupsDataSourceModel

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

	readTimeout := defaultDeviceGroupsReadTimeout
	if !data.Timeouts.IsNull() && !data.Timeouts.IsUnknown() {
		configuredTimeout, timeoutDiags := data.Timeouts.Read(ctx, defaultDeviceGroupsReadTimeout)
		resp.Diagnostics.Append(timeoutDiags...)
		if resp.Diagnostics.HasError() {
			return
		}
		readTimeout = configuredTimeout
	}

	readCtx, cancel := context.WithTimeout(ctx, readTimeout)
	defer cancel()

	filter := buildDeviceGroupFilter(&data)
	groups, err := d.client.GetDeviceGroupsV1(readCtx, nil, filter)
	if err != nil {
		resp.Diagnostics.AddError("Unable to list device groups", err.Error())
		return
	}

	results := make([]DeviceGroupsDataSourceResultModel, 0, len(groups))
	for _, grp := range groups {
		description := types.StringNull()
		if grp.Description != "" {
			description = types.StringValue(grp.Description)
		}

		deviceType := types.StringNull()
		if grp.DeviceType != "" {
			deviceType = types.StringValue(strings.ToLower(grp.DeviceType))
		}

		groupType := types.StringNull()
		if grp.GroupType != "" {
			groupType = types.StringValue(strings.ToLower(grp.GroupType))
		}

		result := DeviceGroupsDataSourceResultModel{
			ID:          types.StringValue(grp.ID),
			Name:        types.StringValue(grp.Name),
			Description: description,
			DeviceType:  deviceType,
			GroupType:   groupType,
			MemberCount: types.Int64Value(int64(grp.MemberCount)),
		}
		results = append(results, result)
	}

	data.DeviceGroups = results
	data.ID = types.StringValue("device_groups")

	tflog.Trace(ctx, "listed device group data source", map[string]interface{}{
		"filter": filter,
		"count":  len(results),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
