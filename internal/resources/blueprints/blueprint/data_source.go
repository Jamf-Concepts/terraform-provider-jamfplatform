// Copyright 2025 Jamf Software LLC.

package blueprint

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/client"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &BlueprintDataSource{}

// NewBlueprintDataSource returns a new instance of BlueprintDataSource.
func NewBlueprintDataSource() datasource.DataSource {
	return &BlueprintDataSource{}
}

// Metadata sets the data source type name for the Terraform provider.
func (d *BlueprintDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_blueprints_blueprint"
}

// Schema sets the Terraform schema for the data source.
func (d *BlueprintDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
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
			"device_groups": schema.SetAttribute{
				Description: "Device groups in scope (unordered).",
				ElementType: types.StringType,
				Computed:    true,
			},
			"component": schema.ListNestedAttribute{
				Description: "Blueprint components.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"identifier": schema.StringAttribute{
							Description: "Component identifier.",
							Computed:    true,
						},
						"configuration": schema.MapAttribute{
							Description: "Component configuration as a map of key-value pairs.",
							ElementType: types.StringType,
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

// Configure sets up the API client for the data source from the provider configuration.
func (d *BlueprintDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

// Read fetches a blueprint by ID or title and populates the Terraform state.
func (d *BlueprintDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data BlueprintDataSourceModel

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

	var bp *client.BlueprintDetail
	var err error
	if !data.ID.IsNull() && data.ID.ValueString() != "" {
		bp, err = d.client.GetBlueprintByID(ctx, data.ID.ValueString())
	} else if !data.Name.IsNull() && data.Name.ValueString() != "" {
		bp, err = d.client.GetBlueprintByName(ctx, data.Name.ValueString())
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

	deviceGroupsSet, _ := types.SetValueFrom(context.Background(), types.StringType, bp.Scope.DeviceGroups)

	var components []ComponentModel
	if len(bp.Steps) > 0 {
		step := bp.Steps[0]
		components = make([]ComponentModel, len(step.Components))
		for i, comp := range step.Components {
			configMap := make(map[string]string)
			if comp.Configuration != nil {
				var jsonObj map[string]interface{}
				if err := json.Unmarshal(comp.Configuration, &jsonObj); err == nil {
					flattenJSON(jsonObj, "", configMap)
				}
			}

			configMapValue, _ := types.MapValueFrom(context.Background(), types.StringType, configMap)
			components[i] = ComponentModel{
				Identifier:    types.StringValue(comp.Identifier),
				Configuration: configMapValue,
			}
		}
	}

	data = BlueprintDataSourceModel{
		ID:              data.ID,
		Name:            types.StringValue(bp.Name),
		BlueprintID:     types.StringValue(bp.ID),
		Description:     types.StringValue(bp.Description),
		Created:         types.StringValue(bp.Created),
		Updated:         types.StringValue(bp.Updated),
		DeploymentState: types.StringValue(bp.DeploymentState.State),
		DeviceGroups:    deviceGroupsSet,
		Components:      components,
	}

	tflog.Trace(ctx, "read a data source")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
