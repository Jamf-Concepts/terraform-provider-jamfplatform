// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

// Package api_clients implements the jamfplatform_pro_api_clients plural data source.
package api_clients

import (
	"context"
	"strconv"
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
// source. Empty: no per-resource floor; the API Integrations endpoints predate
// the provider's overall floor (11.0.0).
const minJamfProVersion = ""

// ApiClientFilterSelectors enumerates the RSQL selectors accepted by the api-integrations endpoint.
var ApiClientFilterSelectors = []string{
	"id",
	"displayName",
}

// ApiClientsDataSource implements the Terraform data source for Jamf Pro API client searches.
type ApiClientsDataSource struct {
	client *pro.Client
}

var _ datasource.DataSource = &ApiClientsDataSource{}

// NewApiClientsDataSource returns a new instance of ApiClientsDataSource.
func NewApiClientsDataSource() datasource.DataSource {
	return &ApiClientsDataSource{}
}

// Metadata sets the data source type name for the Terraform provider.
func (d *ApiClientsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pro_api_clients"
}

// Schema returns the plural data source schema.
func (d *ApiClientsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Search Jamf Pro API clients using optional RSQL filters on `id` and `displayName`. The client secret is never exposed.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Internal identifier for this data source read.",
				Computed:            true,
			},
			"timeouts": timeouts.Attributes(ctx),
			"filter": filters.FilterAttribute(
				filters.SelectorDescription(ApiClientFilterSelectors),
				ApiClientFilterSelectors,
			),
			"api_clients": schema.ListNestedAttribute{
				MarkdownDescription: "API clients matching the supplied filter.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "API client ID assigned by Jamf Pro.",
							Computed:            true,
						},
						"display_name": schema.StringAttribute{
							MarkdownDescription: "API client display name.",
							Computed:            true,
						},
						"api_roles": schema.SetAttribute{
							MarkdownDescription: "The set of API role display names assigned to this client.",
							ElementType:         types.StringType,
							Computed:            true,
						},
						"enabled": schema.BoolAttribute{
							MarkdownDescription: "Whether the client may authenticate.",
							Computed:            true,
						},
						"access_token_lifetime_seconds": schema.Int64Attribute{
							MarkdownDescription: "The lifetime, in seconds, of access tokens issued to this client.",
							Computed:            true,
						},
						"client_id": schema.StringAttribute{
							MarkdownDescription: "The OAuth client identifier assigned by Jamf Pro.",
							Computed:            true,
						},
						"app_type": schema.StringAttribute{
							MarkdownDescription: "Returned by Jamf Pro; not user-settable (`NONE` or `CLIENT_CREDENTIALS`).",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

// Configure wires the Jamf Pro client into the data source via the shared
// providerdata.ConfigurePro helper.
func (d *ApiClientsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	client, diags := providerdata.ConfigurePro(ctx, req.ProviderData, minJamfProVersion, "jamfplatform_pro_api_clients")
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	d.client = client
}

// Read fetches API clients matching the supplied filters and populates state.
func (d *ApiClientsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.client == nil {
		resp.Diagnostics.AddError(
			"Provider not configured",
			"The provider client was not configured. Please ensure the provider block is set up correctly.",
		)
		return
	}

	var data ApiClientsDataSourceModel
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

	filterExpression := filters.BuildRSQLExpression(data.Filters, filters.AllowList(ApiClientFilterSelectors))
	tflog.Debug(ctx, "api clients filter expression", map[string]any{"filter": filterExpression})

	clients, err := d.client.ListApiIntegrationsV1(readCtx, nil, filterExpression)
	if err != nil {
		resp.Diagnostics.AddError("Unable to list Jamf Pro API clients", err.Error())
		return
	}

	results := make([]ApiClientsDataSourceResultModel, 0, len(clients))
	for _, c := range clients {
		scopes, scopeDiags := types.SetValueFrom(ctx, types.StringType, c.AuthorizationScopes)
		resp.Diagnostics.Append(scopeDiags...)
		if resp.Diagnostics.HasError() {
			return
		}
		results = append(results, ApiClientsDataSourceResultModel{
			ID:                         types.StringValue(strconv.Itoa(c.ID)),
			DisplayName:                types.StringValue(c.DisplayName),
			ApiRoles:                   scopes,
			Enabled:                    types.BoolValue(c.Enabled),
			AccessTokenLifetimeSeconds: types.Int64Value(int64(c.AccessTokenLifetimeSeconds)),
			ClientID:                   types.StringValue(c.ClientID),
			AppType:                    types.StringValue(c.AppType),
		})
	}

	data.ApiClients = results
	data.ID = types.StringValue("api_clients")

	tflog.Trace(ctx, "listed Jamf Pro API clients data source", map[string]any{
		"filter": filterExpression,
		"count":  len(results),
	})
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
