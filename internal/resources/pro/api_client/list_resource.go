// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package api_client

import (
	"context"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/pro"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/list"
	listschema "github.com/hashicorp/terraform-plugin-framework/list/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/filters"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/providerdata"
)

var _ list.ListResource = &ApiClientListResource{}
var _ list.ListResourceWithConfigure = &ApiClientListResource{}

// NewApiClientListResource returns a list resource for Jamf Pro API client queries.
func NewApiClientListResource() list.ListResource {
	return &ApiClientListResource{}
}

// ApiClientListResource implements Terraform query list support for Jamf Pro API clients.
type ApiClientListResource struct {
	client *pro.Client
}

// Metadata sets the list resource type name.
func (r *ApiClientListResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pro_api_client"
}

// Configure wires the Jamf Pro client into the list resource via the shared
// providerdata.ConfigurePro helper.
func (r *ApiClientListResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	client, diags := providerdata.ConfigurePro(ctx, req.ProviderData, minJamfProVersion, "jamfplatform_pro_api_client")
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = client
}

// ListResourceConfigSchema describes the supported list filters.
func (r *ApiClientListResource) ListResourceConfigSchema(ctx context.Context, req list.ListResourceSchemaRequest, resp *list.ListResourceSchemaResponse) {
	resp.Schema = listschema.Schema{
		Description: "Searches for Jamf Pro API clients using the same filter clauses as the api_clients data source.",
		Attributes: map[string]listschema.Attribute{
			"filter": filters.ListFilterAttribute(
				filters.SelectorDescription(ApiClientFilterSelectors),
				ApiClientFilterSelectors,
			),
		},
	}
}

// List executes the query and streams API client identities back to Terraform.
// The client secret is never streamed — Jamf Pro does not return it on read.
func (r *ApiClientListResource) List(ctx context.Context, req list.ListRequest, stream *list.ListResultsStream) {
	if r.client == nil {
		stream.Results = list.ListResultsStreamDiagnostics(diag.Diagnostics{
			diag.NewErrorDiagnostic(
				"Unconfigured Provider",
				"The provider has not been configured yet. Re-run the command after `terraform init` completes successfully.",
			),
		})
		return
	}

	var config ApiClientListResourceModel
	diags := req.Config.Get(ctx, &config)
	if diags.HasError() {
		stream.Results = list.ListResultsStreamDiagnostics(diags)
		return
	}

	filterExpression := filters.BuildRSQLExpression(config.Filters, filters.AllowList(ApiClientFilterSelectors))
	tflog.Debug(ctx, "api client list filters", map[string]any{"filter": filterExpression})

	clients, err := r.client.ListApiIntegrationsV1(ctx, nil, filterExpression)
	if err != nil {
		stream.Results = list.ListResultsStreamDiagnostics(diag.Diagnostics{
			diag.NewErrorDiagnostic("Unable to list Jamf Pro API clients", err.Error()),
		})
		return
	}

	maxResults := req.Limit
	if maxResults <= 0 || maxResults > int64(len(clients)) {
		maxResults = int64(len(clients))
	}

	results := make([]list.ListResult, 0, maxResults)

	for _, c := range clients {
		if int64(len(results)) >= maxResults {
			break
		}

		result := req.NewListResult(ctx)
		result.DisplayName = c.DisplayName

		id := types.StringValue(idString(c.ID))
		result.Diagnostics.Append(helpers.SetIdentity(ctx, result.Identity, apiClientIdentityModel{ID: id})...)
		if result.Diagnostics.HasError() {
			stream.Results = list.ListResultsStreamDiagnostics(result.Diagnostics)
			return
		}

		if req.IncludeResource {
			scopes, scopeDiags := types.SetValueFrom(ctx, types.StringType, c.AuthorizationScopes)
			result.Diagnostics.Append(scopeDiags...)
			if result.Diagnostics.HasError() {
				stream.Results = list.ListResultsStreamDiagnostics(result.Diagnostics)
				return
			}
			state := ApiClientResourceModel{
				ID:                         id,
				DisplayName:                types.StringValue(c.DisplayName),
				ApiRoles:                   scopes,
				Enabled:                    types.BoolValue(c.Enabled),
				AccessTokenLifetimeSeconds: types.Int64Value(int64(c.AccessTokenLifetimeSeconds)),
				ClientID:                   types.StringValue(c.ClientID),
				AppType:                    types.StringValue(c.AppType),
				CredentialRotation:         types.StringNull(),
				ClientSecret:               types.StringNull(),
				Timeouts:                   helpers.NewResourceTimeoutsNullValue(apiClientTimeoutAttributeTypes),
			}
			result.Diagnostics.Append(result.Resource.Set(ctx, &state)...)
			if result.Diagnostics.HasError() {
				stream.Results = list.ListResultsStreamDiagnostics(result.Diagnostics)
				return
			}
		}

		results = append(results, result)
	}

	tflog.Debug(ctx, "Listed Jamf Pro API clients", map[string]any{
		"filter":   filterExpression,
		"limit":    req.Limit,
		"returned": len(results),
	})

	if len(results) == 0 {
		stream.Results = list.NoListResults
		return
	}

	stream.Results = func(push func(list.ListResult) bool) {
		for _, result := range results {
			if !push(result) {
				return
			}
		}
	}
}
