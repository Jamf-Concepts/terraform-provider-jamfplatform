// Copyright 2025 Jamf Software LLC.

package baselines

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/client"
)

// Ensure baselinesDataSource implements the DataSource interface
var _ datasource.DataSource = &baselinesDataSource{}

// NewBaselinesDataSource returns a new instance of the baselines data source.
func NewBaselinesDataSource() datasource.DataSource {
	return &baselinesDataSource{}
}

// Configure sets up the API client for the data source from the provider configuration.
func (d *baselinesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *baselinesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cbengine_baselines"
}

// Schema sets the Terraform schema for the data source.
func (d *baselinesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Returns list of the mSCP baselines allowed for the Compliance benchmarks.",
		Attributes: map[string]schema.Attribute{
			"baselines": schema.ListNestedAttribute{
				Description: "List of baselines.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "Unique identifier for the baseline.",
							Computed:    true,
						},
						"baseline_id": schema.StringAttribute{
							Description: "Baseline ID.",
							Computed:    true,
						},
						"title": schema.StringAttribute{
							Description: "Title of the baseline.",
							Computed:    true,
						},
						"description": schema.StringAttribute{
							Description: "Description of the baseline.",
							Computed:    true,
						},
						"rule_count": schema.Int64Attribute{
							Description: "Number of rules in the baseline.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

// Read implements datasource.DataSource for baselinesDataSource. It fetches the list of baselines from the API and sets the state.
func (d *baselinesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.client == nil {
		resp.Diagnostics.AddError(
			"Provider not configured",
			"The provider client was not configured. Please ensure provider block is set up correctly.",
		)
		return
	}

	baselinesResp, err := d.client.GetCBEngineBaselines(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to get baselines",
			err.Error(),
		)
		return
	}

	var baselines []baselineModel
	for _, b := range baselinesResp.Baselines {
		baselines = append(baselines, baselineModel{
			ID:          types.StringValue(b.ID),
			BaselineID:  types.StringValue(b.BaselineID),
			Title:       types.StringValue(b.Title),
			Description: types.StringValue(b.Description),
			RuleCount:   types.Int64Value(b.RuleCount),
		})
	}

	var state baselinesDataSourceModel
	state.Baselines = baselines

	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}
