// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package category

import (
	"context"
	"fmt"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/pro"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/datasource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/providerdata"
)

// CategoryDataSource implements the Terraform data source for Jamf Pro categories.
type CategoryDataSource struct {
	client *pro.Client
}

var _ datasource.DataSource = &CategoryDataSource{}

// NewCategoryDataSource returns a new instance of CategoryDataSource.
func NewCategoryDataSource() datasource.DataSource {
	return &CategoryDataSource{}
}

// Metadata sets the data source type name for the Terraform provider.
func (d *CategoryDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pro_category"
}

// Schema returns the data source schema.
func (d *CategoryDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Look up a Jamf Pro category by ID.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Category ID to look up.",
				Required:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Category display name.",
				Computed:            true,
			},
			"priority": schema.Int64Attribute{
				MarkdownDescription: "Sort priority, 1–20. Lower numbers sort first.",
				Computed:            true,
			},
			"timeouts": timeouts.Attributes(ctx),
		},
	}
}

// Configure wires the Jamf Pro client into the data source. Mirrors the resource Configure
// shape: always fetches the tenant Jamf Pro version (cached), runs the per-resource gate
// when set, and surfaces the provider-floor advisory warning when applicable.
func (d *CategoryDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	pd, ok := req.ProviderData.(*providerdata.Data)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *providerdata.Data, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}
	d.client = pro.New(pd.Client)

	version, err := pd.GetJamfProVersion(ctx)
	if err != nil {
		if minJamfProVersion == "" {
			return
		}
		resp.Diagnostics.AddError(
			"Failed to read Jamf Pro tenant version",
			fmt.Sprintf("jamfplatform_pro_category requires Jamf Pro; could not read version: %s", err),
		)
		return
	}
	if minJamfProVersion != "" {
		resp.Diagnostics.Append(
			helpers.RequireMinJamfProVersion(version, minJamfProVersion, "jamfplatform_pro_category")...,
		)
	}
	if warn := pd.MaybeProviderFloorWarning(); warn != nil {
		resp.Diagnostics.Append(warn)
	}
}

// Read fetches a category by ID and populates Terraform state.
func (d *CategoryDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data CategoryDataSourceModel
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

	if data.ID.IsNull() || data.ID.ValueString() == "" {
		resp.Diagnostics.AddError("Missing ID", "The id attribute must be provided to read a Jamf Pro category.")
		return
	}

	readTimeout, timeoutDiags := helpers.ResolveTimeout(ctx, data.Timeouts.IsNull(), data.Timeouts.IsUnknown(), defaultReadTimeout, data.Timeouts.Read)
	resp.Diagnostics.Append(timeoutDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	readCtx, cancel := context.WithTimeout(ctx, readTimeout)
	defer cancel()

	got, err := d.client.GetCategoryV1(readCtx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to find Jamf Pro category", err.Error())
		return
	}
	assignCategoryDataSourceModel(&data, got)

	tflog.Trace(ctx, "read Jamf Pro category data source", map[string]any{"id": data.ID.ValueString()})
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
