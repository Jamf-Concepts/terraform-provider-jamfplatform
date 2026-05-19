// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

// Package categories implements the jamfplatform_pro_categories plural data source.
package categories

import (
	"context"
	"fmt"
	"time"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/pro"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/datasource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/filters"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/providerdata"
)

const defaultReadTimeout = 90 * time.Second

// minJamfProVersion mirrors the singular category's gate (empty = no gate).
const minJamfProVersion = ""

// CategoryFilterSelectors enumerates the RSQL selectors accepted by the categories endpoint.
var CategoryFilterSelectors = []string{
	"name",
	"priority",
}

// CategoriesDataSource implements the Terraform data source for Jamf Pro category searches.
type CategoriesDataSource struct {
	client *pro.Client
}

var _ datasource.DataSource = &CategoriesDataSource{}

// NewCategoriesDataSource returns a new instance of CategoriesDataSource.
func NewCategoriesDataSource() datasource.DataSource {
	return &CategoriesDataSource{}
}

// Metadata sets the data source type name for the Terraform provider.
func (d *CategoriesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pro_categories"
}

// Schema returns the plural data source schema.
func (d *CategoriesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Search Jamf Pro categories using optional RSQL filters on `name` and `priority`.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Internal identifier for this data source read.",
				Computed:            true,
			},
			"timeouts": timeouts.Attributes(ctx),
			"filter": filters.FilterAttribute(
				filters.SelectorDescription(CategoryFilterSelectors),
				CategoryFilterSelectors,
			),
			"categories": schema.ListNestedAttribute{
				MarkdownDescription: "Categories matching the supplied filter.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "Category ID assigned by Jamf Pro.",
							Computed:            true,
						},
						"name": schema.StringAttribute{
							MarkdownDescription: "Category display name.",
							Computed:            true,
						},
						"priority": schema.Int64Attribute{
							MarkdownDescription: "Sort priority, 1–20.",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

// Configure wires the Jamf Pro client into the data source.
func (d *CategoriesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
			fmt.Sprintf("jamfplatform_pro_categories requires Jamf Pro; could not read version: %s", err),
		)
		return
	}
	if minJamfProVersion != "" {
		resp.Diagnostics.Append(
			helpers.RequireMinJamfProVersion(version, minJamfProVersion, "jamfplatform_pro_categories")...,
		)
	}
	if warn := pd.MaybeProviderFloorWarning(); warn != nil {
		resp.Diagnostics.Append(warn)
	}
}

// Read fetches categories matching the supplied filters and populates state.
func (d *CategoriesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data CategoriesDataSourceModel
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

	readTimeout, timeoutDiags := helpers.ResolveTimeout(ctx, data.Timeouts.IsNull(), data.Timeouts.IsUnknown(), defaultReadTimeout, data.Timeouts.Read)
	resp.Diagnostics.Append(timeoutDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	readCtx, cancel := context.WithTimeout(ctx, readTimeout)
	defer cancel()

	filterExpression := filters.BuildRSQLExpression(data.Filters, filters.AllowList(CategoryFilterSelectors))
	tflog.Debug(ctx, "categories filter expression", map[string]any{"filter": filterExpression})

	cats, err := d.client.ListCategoriesV1(readCtx, nil, filterExpression)
	if err != nil {
		resp.Diagnostics.AddError("Unable to list Jamf Pro categories", err.Error())
		return
	}

	results := make([]CategoriesDataSourceResultModel, 0, len(cats))
	for _, c := range cats {
		results = append(results, CategoriesDataSourceResultModel{
			ID:       helpers.StringPointerValueOrNull(c.ID),
			Name:     types.StringValue(c.Name),
			Priority: types.Int64Value(int64(c.Priority)),
		})
	}

	data.Categories = results
	data.ID = types.StringValue("categories")

	tflog.Trace(ctx, "listed Jamf Pro categories data source", map[string]any{
		"filter": filterExpression,
		"count":  len(results),
	})
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
