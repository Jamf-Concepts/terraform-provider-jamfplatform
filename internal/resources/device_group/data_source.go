// Copyright 2025 Jamf Software LLC.

package device_group

import (
	"context"
	"fmt"
	"strings"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &DeviceGroupDataSource{}

// NewDeviceGroupDataSource returns a new instance of DeviceGroupDataSource.
func NewDeviceGroupDataSource() datasource.DataSource {
	return &DeviceGroupDataSource{}
}

// Metadata sets the data source type name for the Terraform provider.
func (d *DeviceGroupDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_device_group"
}

// Schema sets the Terraform schema for the data source.
func (d *DeviceGroupDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lookup a Jamf device group by ID or name. Requires **Device Group Inventory API** access.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Optional device group Platform ID to query.",
				Optional:            true,
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Optional device group name to query (case-insensitive).",
				Optional:            true,
				Computed:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Device group Description.",
				Computed:            true,
			},
			"device_type": schema.StringAttribute{
				MarkdownDescription: "Optional device type filter. When set, the value is returned in lowercase. Valid values are `computer` and `mobile`.",
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("computer", "mobile"),
				},
			},
			"group_type": schema.StringAttribute{
				MarkdownDescription: "Optional group type filter. When set, the value is returned in lowercase. Valid values are `static` and `smart`.",
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("static", "smart"),
				},
			},
			"member_count": schema.Int64Attribute{
				MarkdownDescription: "Number of members in the group.",
				Computed:            true,
			},
			"members": schema.SetAttribute{
				MarkdownDescription: "Devices currently assigned to the group (Jamf Pro Management IDs).",
				Computed:            true,
				ElementType:         types.StringType,
			},
		},
		Blocks: map[string]schema.Block{
			"criteria": schema.ListNestedBlock{
				MarkdownDescription: "Smart-group criteria returned by the API.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"order": schema.Int64Attribute{
							MarkdownDescription: "Server-evaluated order for the criterion.",
							Computed:            true,
						},
						"criteria": schema.StringAttribute{
							MarkdownDescription: "Inventory attribute used in the criterion.",
							Computed:            true,
						},
						"operator": schema.StringAttribute{
							MarkdownDescription: "Comparison operator.",
							Computed:            true,
						},
						"value": schema.StringAttribute{
							MarkdownDescription: "Comparison value, when applicable.",
							Computed:            true,
						},
						"and_or": schema.StringAttribute{
							MarkdownDescription: "Join type between criteria (AND/OR).",
							Computed:            true,
						},
						"has_opening_parenthesis": schema.BoolAttribute{
							MarkdownDescription: "Whether the criterion starts a parenthetical expression.",
							Computed:            true,
						},
						"has_closing_parenthesis": schema.BoolAttribute{
							MarkdownDescription: "Whether the criterion ends a parenthetical expression.",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

// Configure sets up the API client for the data source from the provider configuration.
func (d *DeviceGroupDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

// Read fetches a device group by ID or name and populates the Terraform state.
func (d *DeviceGroupDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data DeviceGroupDataSourceModel

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
	if !data.ID.IsNull() && data.ID.ValueString() != "" {
		lookupID = data.ID.ValueString()
	}

	lookupName := ""
	if !data.Name.IsNull() && data.Name.ValueString() != "" {
		lookupName = data.Name.ValueString()
	}

	desiredDeviceType := ""
	if !data.DeviceType.IsNull() && data.DeviceType.ValueString() != "" {
		desiredDeviceType = strings.ToLower(data.DeviceType.ValueString())
	}

	desiredGroupType := ""
	if !data.GroupType.IsNull() && data.GroupType.ValueString() != "" {
		desiredGroupType = strings.ToLower(data.GroupType.ValueString())
	}

	if lookupID == "" && lookupName == "" {
		resp.Diagnostics.AddError(
			"Missing query arguments",
			"Either id or name must be provided to read a device group.",
		)
		return
	}

	if lookupID != "" && lookupName != "" {
		resp.Diagnostics.AddError(
			"Conflicting query arguments",
			"Only one of id or name can be set when reading a device group.",
		)
		return
	}

	var (
		grp *client.DeviceGroupReadRepresentationV1
		err error
	)

	if lookupID != "" {
		grp, err = d.client.GetDeviceGroupByIDV1(ctx, lookupID)
	} else {
		grp, err = d.lookupDeviceGroupByFilters(ctx, lookupName, desiredDeviceType, desiredGroupType)
	}

	if err != nil {
		resp.Diagnostics.AddError("Unable to find device group", err.Error())
		return
	}

	if desiredDeviceType != "" && !strings.EqualFold(grp.DeviceType, desiredDeviceType) {
		resp.Diagnostics.AddError(
			"Device type filter mismatch",
			fmt.Sprintf("Device group %s has device_type %q, expected %q", grp.ID, grp.DeviceType, desiredDeviceType),
		)
		return
	}

	if desiredGroupType != "" && !strings.EqualFold(grp.GroupType, desiredGroupType) {
		resp.Diagnostics.AddError(
			"Group type filter mismatch",
			fmt.Sprintf("Device group %s has group_type %q, expected %q", grp.ID, grp.GroupType, desiredGroupType),
		)
		return
	}

	members, err := d.client.GetDeviceGroupMembersV1(ctx, grp.ID)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read device group members", err.Error())
		return
	}

	setMembers, diags := types.SetValueFrom(ctx, types.StringType, members)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

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

	data = DeviceGroupDataSourceModel{
		ID:          types.StringValue(grp.ID),
		Name:        types.StringValue(grp.Name),
		Description: description,
		DeviceType:  deviceType,
		GroupType:   groupType,
		Criteria:    flattenDeviceGroupCriteria(grp.Criteria, nil),
		Members:     setMembers,
		MemberCount: types.Int64Value(int64(grp.MemberCount)),
	}

	tflog.Trace(ctx, "read device group data source", map[string]interface{}{
		"id": grp.ID,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// lookupDeviceGroupByFilters searches for a device group by name and optional filters.
func (d *DeviceGroupDataSource) lookupDeviceGroupByFilters(ctx context.Context, name, deviceType, groupType string) (*client.DeviceGroupReadRepresentationV1, error) {
	clauses := []string{}
	if name != "" {
		clauses = append(clauses, fmt.Sprintf(`name=="%s"`, escapeFilterValue(name)))
	}
	if deviceType != "" {
		clauses = append(clauses, fmt.Sprintf(`deviceType=="%s"`, escapeFilterValue(strings.ToUpper(deviceType))))
	}
	if groupType != "" {
		clauses = append(clauses, fmt.Sprintf(`groupType=="%s"`, escapeFilterValue(strings.ToUpper(groupType))))
	}
	filter := strings.Join(clauses, " and ")

	groups, err := d.client.GetDeviceGroupsV1(ctx, nil, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to search device groups: %w", err)
	}

	switch len(groups) {
	case 0:
		return nil, fmt.Errorf("no device groups matched filter %q", filter)
	case 1:
		group := groups[0]
		needsFullFetch := group.GroupType == "SMART" && len(group.Criteria) == 0
		if needsFullFetch {
			return d.client.GetDeviceGroupByIDV1(ctx, group.ID)
		}
		return &client.DeviceGroupReadRepresentationV1{
			ID:          group.ID,
			Name:        group.Name,
			Description: group.Description,
			DeviceType:  group.DeviceType,
			GroupType:   group.GroupType,
			MemberCount: group.MemberCount,
			Criteria:    group.Criteria,
		}, nil
	default:
		return nil, fmt.Errorf("multiple device groups matched filter %q; please refine the query or use id", filter)
	}
}

func escapeFilterValue(value string) string {
	return strings.ReplaceAll(value, "\"", `\\"`)
}
