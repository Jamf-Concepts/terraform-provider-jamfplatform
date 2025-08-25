// Copyright 2025 Jamf Software LLC.

package component

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/client"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &ComponentDataSource{}

// NewComponentDataSource returns a new instance of ComponentDataSource.
func NewComponentDataSource() datasource.DataSource {
	return &ComponentDataSource{}
}

// Metadata sets the data source type name for the Terraform provider.
func (d *ComponentDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_blueprints_component"
}

// Schema sets the Terraform schema for the data source.
func (d *ComponentDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Returns a blueprint component by identifier.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The component identifier to fetch.",
				Required:    true,
			},
			"identifier": schema.StringAttribute{
				Description: "Component identifier.",
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: "Component name.",
				Computed:    true,
			},
			"description": schema.StringAttribute{
				Description: "Component description.",
				Computed:    true,
			},
			"supported_os": schema.MapAttribute{
				Description: "Supported operating systems with their versions.",
				ElementType: types.ListType{
					ElemType: types.StringType,
				},
				Computed: true,
			},
		},
	}
}

// Configure sets up the API client for the data source from the provider configuration.
func (d *ComponentDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

// Read fetches a component by identifier and populates the Terraform state.
func (d *ComponentDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ComponentDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if d.client == nil {
		resp.Diagnostics.AddError(
			"Provider not configured",
			"The provider client was not configured. Please ensure provider block is set up correctly.",
		)
		return
	}

	if data.ID.IsNull() || data.ID.ValueString() == "" {
		resp.Diagnostics.AddError(
			"Missing Required Attribute",
			"The 'id' attribute must be set to look up a component.",
		)
		return
	}

	comp, err := d.client.GetBlueprintComponentByID(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to get component",
			err.Error(),
		)
		return
	}

	supportedOsAttrType := types.ListType{
		ElemType: types.StringType,
	}

	var supportedOsMap attr.Value
	if len(comp.Meta.SupportedOs) == 0 {
		supportedOsMap = types.MapNull(supportedOsAttrType)
	} else {
		supportedOsMapVals := make(map[string]attr.Value)
		for osFamily, versions := range comp.Meta.SupportedOs {
			osVersionVals := make([]attr.Value, len(versions))
			for i, v := range versions {
				osVersionVals[i] = types.StringValue(v.Version)
			}
			supportedOsList, _ := types.ListValue(types.StringType, osVersionVals)
			supportedOsMapVals[osFamily] = supportedOsList
		}
		supportedOsMap, _ = types.MapValue(supportedOsAttrType, supportedOsMapVals)
	}

	state := ComponentDataSourceModel{
		ID:          data.ID,
		Identifier:  types.StringValue(comp.Identifier),
		Name:        types.StringValue(comp.Name),
		Description: types.StringValue(comp.Description),
		SupportedOs: supportedOsMap.(types.Map),
	}

	tflog.Trace(ctx, "read a data source")

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
