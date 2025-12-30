// Copyright 2025 Jamf Software LLC.

package baselines

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/datasource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/client"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &BaselinesDataSource{}

// NewBaselinesDataSource returns a new instance of BaselinesDataSource.
func NewBaselinesDataSource() datasource.DataSource {
	return &BaselinesDataSource{}
}

// Metadata sets the data source type name for the Terraform provider.
func (d *BaselinesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cbengine_baselines"
}

const defaultReadTimeout = 90 * time.Second

// Schema sets the Terraform schema for the data source.
func (d *BaselinesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Returns list of the mSCP baselines allowed for the Compliance benchmarks. Requires **Compliance Benchmarks API** access.",
		Attributes: map[string]schema.Attribute{
			"timeouts": timeouts.Attributes(ctx),
			"baselines": schema.ListNestedAttribute{
				MarkdownDescription: "List of baselines.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "Unique identifier for the baseline.",
							Computed:            true,
						},
						"baseline_id": schema.StringAttribute{
							MarkdownDescription: "Baseline ID.",
							Computed:            true,
						},
						"title": schema.StringAttribute{
							MarkdownDescription: "Title of the baseline.",
							Computed:            true,
						},
						"description": schema.StringAttribute{
							MarkdownDescription: "Description of the baseline.",
							Computed:            true,
						},
						"rule_count": schema.Int64Attribute{
							MarkdownDescription: "Number of rules in the baseline.",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

// Configure sets up the API client for the data source from the provider configuration.
func (d *BaselinesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

// Read implements datasource.DataSource for BaselinesDataSource. It fetches the list of baselines from the API and sets the state.
func (d *BaselinesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data BaselinesDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	timeoutsValue := data.Timeouts

	readTimeout, timeoutDiags := helpers.ResolveTimeout(ctx, data.Timeouts.IsNull(), data.Timeouts.IsUnknown(), defaultReadTimeout, data.Timeouts.Read)
	resp.Diagnostics.Append(timeoutDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	readCtx, cancel := context.WithTimeout(ctx, readTimeout)
	defer cancel()

	if d.client == nil {
		resp.Diagnostics.AddError(
			"Provider not configured",
			"The provider client was not configured. Please ensure provider block is set up correctly.",
		)
		return
	}

	baselinesResp, err := d.client.GetCBEngineBaselinesV1(readCtx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to get baselines",
			err.Error(),
		)
		return
	}

	var baselines []BaselineModel
	for _, b := range baselinesResp.Baselines {
		baselines = append(baselines, BaselineModel{
			ID:          types.StringValue(b.ID),
			BaselineID:  types.StringValue(b.BaselineID),
			Title:       types.StringValue(b.Title),
			Description: types.StringValue(b.Description),
			RuleCount:   types.Int64Value(b.RuleCount),
		})
	}

	data.Baselines = baselines
	data.Timeouts = timeoutsValue

	tflog.Trace(ctx, "read a data source")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
