// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

// Package buildings implements the jamfplatform_pro_buildings plural data source.
package buildings

import (
	"context"
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

// minJamfProVersion is the minimum Jamf Pro tenant version required by this data
// source. Empty: no per-resource floor; the buildings endpoint has been stable since
// well before the provider's overall floor (11.0.0). Provider-level advisory warning
// still applies via providerdata.ConfigurePro.
const minJamfProVersion = ""

// BuildingFilterSelectors enumerates the RSQL selectors accepted by the buildings endpoint.
// RSQL selectors retain API-native (camelCase) spelling per STYLE_GUIDE.md §Schema Guidelines.
var BuildingFilterSelectors = []string{
	"id",
	"name",
	"city",
	"country",
	"stateProvince",
	"streetAddress1",
	"streetAddress2",
	"zipPostalCode",
}

// BuildingsDataSource implements the Terraform data source for Jamf Pro building searches.
type BuildingsDataSource struct {
	client *pro.Client
}

var (
	_ datasource.DataSource              = &BuildingsDataSource{}
	_ datasource.DataSourceWithConfigure = &BuildingsDataSource{}
)

// NewBuildingsDataSource returns a new instance of BuildingsDataSource.
func NewBuildingsDataSource() datasource.DataSource {
	return &BuildingsDataSource{}
}

// Metadata sets the data source type name for the Terraform provider.
func (d *BuildingsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pro_buildings"
}

// Schema returns the plural data source schema.
func (d *BuildingsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Search Jamf Pro buildings using optional RSQL filters.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Internal identifier for this data source read.",
				Computed:            true,
			},
			"timeouts": timeouts.Attributes(ctx),
			"filter": filters.FilterAttribute(
				filters.SelectorDescription(BuildingFilterSelectors),
				BuildingFilterSelectors,
			),
			"buildings": schema.ListNestedAttribute{
				MarkdownDescription: "Buildings matching the supplied filter.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":               schema.StringAttribute{MarkdownDescription: "Building ID assigned by Jamf Pro.", Computed: true},
						"name":             schema.StringAttribute{MarkdownDescription: "Building display name.", Computed: true},
						"city":             schema.StringAttribute{MarkdownDescription: "City the building is located in.", Computed: true},
						"country":          schema.StringAttribute{MarkdownDescription: "Country the building is located in.", Computed: true},
						"state_province":   schema.StringAttribute{MarkdownDescription: "State, province, or administrative region.", Computed: true},
						"street_address_1": schema.StringAttribute{MarkdownDescription: "Primary street address line.", Computed: true},
						"street_address_2": schema.StringAttribute{MarkdownDescription: "Secondary street address line.", Computed: true},
						"zip_postal_code":  schema.StringAttribute{MarkdownDescription: "Zip or postal code.", Computed: true},
					},
				},
			},
		},
	}
}

// Configure wires the Jamf Pro client into the data source via the shared
// providerdata.ConfigurePro helper.
func (d *BuildingsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	client, diags := providerdata.ConfigurePro(ctx, req.ProviderData, minJamfProVersion, "jamfplatform_pro_buildings")
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	d.client = client
}

// Read fetches buildings matching the supplied filters and populates state.
func (d *BuildingsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.client == nil {
		resp.Diagnostics.AddError(
			"Provider not configured",
			"The provider client was not configured. Please ensure the provider block is set up correctly.",
		)
		return
	}

	var data BuildingsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	readTimeout, timeoutDiags := helpers.ResolveTimeout(ctx, data.Timeouts.IsNull(), data.Timeouts.IsUnknown(), defaultReadTimeout, data.Timeouts.Read)
	resp.Diagnostics.Append(timeoutDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	readCtx, cancel := context.WithTimeout(ctx, readTimeout)
	defer cancel()

	filterExpression := filters.BuildRSQLExpression(data.Filters, filters.AllowList(BuildingFilterSelectors))
	tflog.Debug(ctx, "buildings filter expression", map[string]any{"filter": filterExpression})

	list, err := d.client.ListBuildingsV1(readCtx, nil, filterExpression)
	if err != nil {
		resp.Diagnostics.AddError("Unable to list Jamf Pro buildings", err.Error())
		return
	}

	results := make([]BuildingsDataSourceResultModel, 0, len(list))
	for _, b := range list {
		results = append(results, BuildingsDataSourceResultModel{
			ID:             helpers.StringPointerValueOrNull(b.ID),
			Name:           types.StringValue(b.Name),
			City:           helpers.StringPointerValueOrNull(b.City),
			Country:        helpers.StringPointerValueOrNull(b.Country),
			StateProvince:  helpers.StringPointerValueOrNull(b.StateProvince),
			StreetAddress1: helpers.StringPointerValueOrNull(b.StreetAddress1),
			StreetAddress2: helpers.StringPointerValueOrNull(b.StreetAddress2),
			ZipPostalCode:  helpers.StringPointerValueOrNull(b.ZipPostalCode),
		})
	}

	data.Buildings = results
	data.ID = types.StringValue("buildings")

	tflog.Trace(ctx, "listed Jamf Pro buildings data source", map[string]any{
		"filter": filterExpression,
		"count":  len(results),
	})
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
