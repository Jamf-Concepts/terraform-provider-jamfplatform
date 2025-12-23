// Copyright 2025 Jamf Software LLC.

package benchmarks

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/datasource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/client"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &BenchmarksDataSource{}

// NewBenchmarksDataSource instantiates the benchmarks listing data source.
func NewBenchmarksDataSource() datasource.DataSource {
	return &BenchmarksDataSource{}
}

// Metadata sets the data source type name for the Terraform provider.
func (d *BenchmarksDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cbengine_benchmarks"
}

const defaultReadTimeout = 90 * time.Second

// Schema defines the Terraform schema for listing CBEngine benchmarks.
func (d *BenchmarksDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Returns all Compliance Benchmarks from Jamf Pro. Requires **Compliance Benchmarks API** access.",
		Attributes: map[string]schema.Attribute{
			"timeouts": timeouts.Attributes(ctx),
			"id": schema.StringAttribute{
				MarkdownDescription: "Internal identifier for this data source read.",
				Computed:            true,
			},
			"benchmarks": schema.ListNestedAttribute{
				MarkdownDescription: "CBEngine benchmarks returned by GetCBEngineBenchmarksV2.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "Benchmark identifier.",
							Computed:            true,
						},
						"title": schema.StringAttribute{
							MarkdownDescription: "Benchmark title.",
							Computed:            true,
						},
						"description": schema.StringAttribute{
							MarkdownDescription: "Benchmark description, if provided.",
							Computed:            true,
						},
						"update_available": schema.BoolAttribute{
							MarkdownDescription: "Indicates whether an update is available for the benchmark.",
							Computed:            true,
						},
						"sync_state": schema.StringAttribute{
							MarkdownDescription: "Current synchronization state reported by CBEngine.",
							Computed:            true,
						},
						"target_device_groups": schema.ListAttribute{
							MarkdownDescription: "Device groups targeted by the benchmark.",
							ElementType:         types.StringType,
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

// Configure wires the provider client into the data source.
func (d *BenchmarksDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			"Expected *client.Client. Please report this issue to the provider developers.",
		)
		return
	}

	d.client = client
}

// Read invokes GetCBEngineBenchmarksV2 and maps the response into Terraform state.
func (d *BenchmarksDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data BenchmarksDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	timeoutsValue := data.Timeouts

	readTimeout := defaultReadTimeout
	if !data.Timeouts.IsNull() && !data.Timeouts.IsUnknown() {
		configuredTimeout, timeoutDiags := data.Timeouts.Read(ctx, defaultReadTimeout)
		resp.Diagnostics.Append(timeoutDiags...)
		if resp.Diagnostics.HasError() {
			return
		}
		readTimeout = configuredTimeout
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

	benchmarks, err := d.client.GetCBEngineBenchmarksV2(readCtx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to list CBEngine benchmarks", err.Error())
		return
	}

	entries := make([]BenchmarkListItem, 0, len(benchmarks.Benchmarks))
	for _, bench := range benchmarks.Benchmarks {
		targetGroups := types.ListNull(types.StringType)
		if len(bench.Target.DeviceGroups) > 0 {
			values := make([]attr.Value, len(bench.Target.DeviceGroups))
			for i, group := range bench.Target.DeviceGroups {
				values[i] = stringValueOrNull(group)
			}
			targetGroups, _ = types.ListValue(types.StringType, values)
		}

		entries = append(entries, BenchmarkListItem{
			ID:                 stringValueOrNull(bench.ID),
			Title:              stringValueOrNull(bench.Title),
			Description:        stringValueOrNull(bench.Description),
			UpdateAvailable:    types.BoolValue(bench.UpdateAvailable),
			SyncState:          stringValueOrNull(bench.SyncState),
			TargetDeviceGroups: targetGroups,
		})
	}

	data.ID = types.StringValue("cbengine_benchmarks")
	data.Benchmarks = entries
	data.Timeouts = timeoutsValue

	tflog.Trace(ctx, "read cbengine benchmarks data source", map[string]interface{}{
		"count": len(entries),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
