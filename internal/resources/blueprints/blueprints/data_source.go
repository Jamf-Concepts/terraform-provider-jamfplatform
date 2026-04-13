// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package blueprints

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &BlueprintsDataSource{}

// NewBlueprintsDataSource returns a new instance of BlueprintsDataSource.
func NewBlueprintsDataSource() datasource.DataSource {
	return &BlueprintsDataSource{}
}

// Metadata sets the data source type name for the Terraform provider.
func (d *BlueprintsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_blueprints"
}

// Schema sets the Terraform schema for listing blueprints.
func (d *BlueprintsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Returns all Jamf blueprints from Jamf Pro with an optional case-insensitive search filter. Requires **Blueprints API** access.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Internal identifier for this data source read.",
				Computed:            true,
			},
			"search": schema.StringAttribute{
				MarkdownDescription: "Optional substring to match against blueprint name or description (case-insensitive).",
				Optional:            true,
			},
			"blueprints": schema.ListNestedAttribute{
				MarkdownDescription: "Blueprints that matched the optional search filter.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "Blueprint identifier.",
							Computed:            true,
						},
						"name": schema.StringAttribute{
							MarkdownDescription: "Blueprint name.",
							Computed:            true,
						},
						"description": schema.StringAttribute{
							MarkdownDescription: "Blueprint description.",
							Computed:            true,
						},
						"created": schema.StringAttribute{
							MarkdownDescription: "Creation timestamp (RFC3339).",
							Computed:            true,
						},
						"updated": schema.StringAttribute{
							MarkdownDescription: "Last update timestamp (RFC3339).",
							Computed:            true,
						},
						"deployment_state": schema.StringAttribute{
							MarkdownDescription: "Current deployment state.",
							Computed:            true,
						},
						"last_deployment_state": schema.StringAttribute{
							MarkdownDescription: "State of the most recent deployment, if any.",
							Computed:            true,
						},
						"last_deployment_started": schema.StringAttribute{
							MarkdownDescription: "Timestamp when the most recent deployment started, if available.",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

// Configure sets up the API client for the data source from the provider configuration.
func (d *BlueprintsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*jamfplatform.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			"Expected *jamfplatform.Client. Please report this issue to the provider developers.",
		)
		return
	}

	d.client = client
}

// Read fetches all blueprints (optionally filtered) and populates the Terraform state.
func (d *BlueprintsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data BlueprintsDataSourceModel

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

	searchTerm := ""
	if helpers.IsConfiguredValue(data.Search) {
		searchTerm = strings.TrimSpace(data.Search.ValueString())
	}
	searchLower := strings.ToLower(searchTerm)

	blueprints, err := d.client.ListBlueprints(ctx, nil, "")
	if err != nil {
		resp.Diagnostics.AddError("Unable to list blueprints", err.Error())
		return
	}

	entries := make([]BlueprintListItem, 0, len(blueprints))
	for _, bp := range blueprints {
		if searchLower != "" {
			nameMatch := strings.Contains(strings.ToLower(bp.Name), searchLower)
			descMatch := strings.Contains(strings.ToLower(helpers.DerefString(bp.Description)), searchLower)
			if !nameMatch && !descMatch {
				continue
			}
		}

		deployState := ""
		if bp.DeploymentState != nil {
			deployState = bp.DeploymentState.State
		}

		item := BlueprintListItem{
			ID:                    types.StringValue(bp.ID),
			Name:                  helpers.StringValueOrNull(bp.Name),
			Description:           helpers.StringPointerValueOrNull(bp.Description),
			Created:               helpers.StringValueOrNull(bp.Created),
			Updated:               helpers.StringValueOrNull(bp.Updated),
			DeploymentState:       helpers.StringValueOrNull(deployState),
			LastDeploymentState:   types.StringNull(),
			LastDeploymentStarted: types.StringNull(),
		}

		if bp.DeploymentState != nil && bp.DeploymentState.LastDeployment != nil {
			item.LastDeploymentState = helpers.StringValueOrNull(bp.DeploymentState.LastDeployment.State)
			item.LastDeploymentStarted = helpers.StringValueOrNull(bp.DeploymentState.LastDeployment.Started)
		}

		entries = append(entries, item)
	}

	data.ID = types.StringValue("blueprints")
	if searchTerm == "" {
		data.Search = types.StringNull()
	} else {
		data.Search = types.StringValue(searchTerm)
	}
	data.Blueprints = entries

	tflog.Trace(ctx, "read blueprints data source", map[string]any{
		"search": searchTerm,
		"count":  len(entries),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
