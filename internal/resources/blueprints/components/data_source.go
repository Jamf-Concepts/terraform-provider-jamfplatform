// Copyright 2025 Jamf Software LLC.

package components

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/client"
)

// NewComponentsDataSource returns a new instance of componentsDataSource.
func NewComponentsDataSource() datasource.DataSource {
	return &componentsDataSource{}
}

type componentsDataSource struct {
	client *client.Client
}

// Configure sets up the API client for the data source from the provider configuration.
func (d *componentsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *componentsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_blueprints_components"
}

// Schema sets the Terraform schema for the data source.
func (d *componentsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Returns all available blueprint components.",
		Attributes: map[string]schema.Attribute{
			"components": schema.ListNestedAttribute{
				Description: "List of all blueprint components.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
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
						"meta": schema.SingleNestedAttribute{
							Description: "Component metadata.",
							Computed:    true,
							Attributes: map[string]schema.Attribute{
								"supported_os": schema.MapAttribute{
									Description: "Supported operating systems with their versions.",
									ElementType: types.ListType{
										ElemType: types.ObjectType{
											AttrTypes: map[string]attr.Type{
												"version": types.StringType,
											},
										},
									},
									Computed: true,
								},
							},
						},
					},
				},
			},
		},
	}
}

// Read fetches all components and populates the Terraform state.
func (d *componentsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config componentsDataSourceModel
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

	components, err := d.client.GetBlueprintComponents(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to get components",
			err.Error(),
		)
		return
	}

	// Convert components to Terraform types
	var componentsList []componentListModel
	for _, comp := range components {
		supportedOsAttrType := types.ListType{
			ElemType: types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"version": types.StringType,
				},
			},
		}

		metaAttrType := map[string]attr.Type{
			"supported_os": types.MapType{
				ElemType: supportedOsAttrType,
			},
		}

		var meta types.Object
		if len(comp.Meta.SupportedOs) == 0 {
			meta = types.ObjectNull(metaAttrType)
		} else {
			supportedOsMapVals := make(map[string]attr.Value)
			for osFamily, versions := range comp.Meta.SupportedOs {
				osVersionVals := make([]attr.Value, len(versions))
				for i, v := range versions {
					osVersionVals[i], _ = types.ObjectValue(
						map[string]attr.Type{
							"version": types.StringType,
						},
						map[string]attr.Value{
							"version": types.StringValue(v.Version),
						},
					)
				}
				supportedOsList, _ := types.ListValue(
					types.ObjectType{
						AttrTypes: map[string]attr.Type{
							"version": types.StringType,
						},
					},
					osVersionVals,
				)
				supportedOsMapVals[osFamily] = supportedOsList
			}

			supportedOsMap, _ := types.MapValue(supportedOsAttrType, supportedOsMapVals)
			meta, _ = types.ObjectValue(
				metaAttrType,
				map[string]attr.Value{
					"supported_os": supportedOsMap,
				},
			)
		}

		componentsList = append(componentsList, componentListModel{
			Identifier:  types.StringValue(comp.Identifier),
			Name:        types.StringValue(comp.Name),
			Description: types.StringValue(comp.Description),
			Meta:        meta,
		})
	}

	state := componentsDataSourceModel{
		Components: componentsList,
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}
