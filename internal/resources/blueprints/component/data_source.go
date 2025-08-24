// Copyright 2025 Jamf Software LLC.

package component

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/client"
)

// NewComponentDataSource returns a new instance of componentDataSource.
func NewComponentDataSource() datasource.DataSource {
	return &componentDataSource{}
}

type componentDataSource struct {
	client *client.Client
}

// Configure sets up the API client for the data source from the provider configuration.
func (d *componentDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *componentDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_blueprints_component"
}

// Schema sets the Terraform schema for the data source.
func (d *componentDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
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

// Read fetches a component by identifier and populates the Terraform state.
func (d *componentDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config componentDataSourceModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
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

	if config.ID.IsNull() || config.ID.ValueString() == "" {
		resp.Diagnostics.AddError(
			"Missing Required Attribute",
			"The 'id' attribute must be set to look up a component.",
		)
		return
	}

	comp, err := d.client.GetBlueprintComponentByID(ctx, config.ID.ValueString())
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

	state := componentDataSourceModel{
		ID:          config.ID,
		Identifier:  types.StringValue(comp.Identifier),
		Name:        types.StringValue(comp.Name),
		Description: types.StringValue(comp.Description),
		SupportedOs: supportedOsMap.(types.Map),
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}
