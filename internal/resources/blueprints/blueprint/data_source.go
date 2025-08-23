// Copyright 2025 Jamf Software LLC.

package blueprint

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/client"
)

// NewBlueprintDataSource returns a new instance of blueprintDataSource.
func NewBlueprintDataSource() datasource.DataSource {
	return &blueprintDataSource{}
}

type blueprintDataSource struct {
	client *client.Client
}

// Configure sets up the API client for the data source from the provider configuration.
func (d *blueprintDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *blueprintDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_blueprints_blueprint"
}

// Schema sets the Terraform schema for the data source.
func (d *blueprintDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Returns a blueprint by ID or name.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The blueprint ID to fetch. Optional if name is set.",
				Optional:    true,
			},
			"name": schema.StringAttribute{
				Description: "The blueprint name to fetch. Optional if id is set.",
				Optional:    true,
			},
			"blueprint_id": schema.StringAttribute{
				Description: "Blueprint ID.",
				Computed:    true,
			},
			"description": schema.StringAttribute{
				Description: "Description.",
				Computed:    true,
			},
			"created": schema.StringAttribute{
				Description: "Created at (RFC3339).",
				Computed:    true,
			},
			"updated": schema.StringAttribute{
				Description: "Updated at (RFC3339).",
				Computed:    true,
			},
			"deployment_state": schema.StringAttribute{
				Description: "Deployment state.",
				Computed:    true,
			},
			"device_groups": schema.ListAttribute{
				Description: "Device groups in scope.",
				ElementType: types.StringType,
				Computed:    true,
			},
			"steps": schema.ListNestedAttribute{
				Description: "Blueprint steps.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Description: "Step name.",
							Computed:    true,
						},
						"components": schema.ListNestedAttribute{
							Description: "Step components.",
							Computed:    true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"identifier": schema.StringAttribute{
										Description: "Component identifier.",
										Computed:    true,
									},
									"configuration": schema.StringAttribute{
										Description: "Component configuration.",
										Computed:    true,
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

// Read fetches a blueprint by ID or title and populates the Terraform state.
func (d *blueprintDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config blueprintDataSourceModel
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

	var bp *client.BlueprintDetail
	var err error
	if !config.ID.IsNull() && config.ID.ValueString() != "" {
		bp, err = d.client.GetBlueprintByID(ctx, config.ID.ValueString())
	} else if !config.Name.IsNull() && config.Name.ValueString() != "" {
		bp, err = d.client.GetBlueprintByName(ctx, config.Name.ValueString())
	} else {
		resp.Diagnostics.AddError(
			"Missing Required Attribute",
			"Either 'id' or 'name' must be set to look up a blueprint.",
		)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to get blueprint",
			err.Error(),
		)
		return
	}

	var deviceGroups []types.String
	for _, g := range bp.Scope.DeviceGroups {
		deviceGroups = append(deviceGroups, types.StringValue(g))
	}

	var steps []stepModel
	for _, s := range bp.Steps {
		var components []componentModel
		for _, c := range s.Components {
			var configStr string
			if c.Configuration != nil {
				configStr = string(c.Configuration)
			}

			components = append(components, componentModel{
				Identifier:    types.StringValue(c.Identifier),
				Configuration: types.StringValue(configStr),
			})
		}
		steps = append(steps, stepModel{
			Name:       types.StringValue(s.Name),
			Components: components,
		})
	}

	state := blueprintDataSourceModel{
		ID:              config.ID,
		Name:            types.StringValue(bp.Name),
		BlueprintID:     types.StringValue(bp.ID),
		Description:     types.StringValue(bp.Description),
		Created:         types.StringValue(bp.Created),
		Updated:         types.StringValue(bp.Updated),
		DeploymentState: types.StringValue(bp.DeploymentState.State),
		DeviceGroups:    deviceGroups,
		Steps:           steps,
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}
