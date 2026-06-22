// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package pki_json_web_token_configuration

import (
	"context"
	"time"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/proclassic"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/list"
	listschema "github.com/hashicorp/terraform-plugin-framework/list/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/filters"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/providerdata"
)

// defaultListTimeout caps how long the list operation waits on the classic
// JSON Web Token configurations endpoint.
const defaultListTimeout = 90 * time.Second

var (
	_ list.ListResource              = &JSONWebTokenConfigurationListResource{}
	_ list.ListResourceWithConfigure = &JSONWebTokenConfigurationListResource{}
)

// NewJSONWebTokenConfigurationListResource returns a list resource for Jamf Pro
// JSON Web Token configuration queries.
func NewJSONWebTokenConfigurationListResource() list.ListResource {
	return &JSONWebTokenConfigurationListResource{}
}

// JSONWebTokenConfigurationListResource implements Terraform query list support
// for Jamf Pro JSON Web Token configurations. The classic endpoint has no
// server-side filtering — the optional `filter` block is applied client-side
// via filters.ApplyClassicFilter.
type JSONWebTokenConfigurationListResource struct {
	client *proclassic.Client
}

// Metadata sets the list resource type name.
func (r *JSONWebTokenConfigurationListResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pro_pki_json_web_token_configuration"
}

// Configure wires the Jamf ProClassic client into the list resource.
func (r *JSONWebTokenConfigurationListResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	client, diags := providerdata.ConfigureProClassic(ctx, req.ProviderData, minJamfProVersion, "jamfplatform_pro_pki_json_web_token_configuration")
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = client
}

// ListResourceConfigSchema describes the supported list filters.
func (r *JSONWebTokenConfigurationListResource) ListResourceConfigSchema(ctx context.Context, req list.ListResourceSchemaRequest, resp *list.ListResourceSchemaResponse) {
	resp.Schema = listschema.Schema{
		Description: "Lists Jamf Pro JSON Web Token configurations. Supply an optional case-insensitive `name_substring` filter applied locally after the full list is fetched. List entries surface as identity-only (id and display name); full detail requires a per-record read.",
		Attributes: map[string]listschema.Attribute{
			"filter": filters.ClassicListFilterAttribute(),
		},
	}
}

// List executes the query and streams record identities back to Terraform.
func (r *JSONWebTokenConfigurationListResource) List(ctx context.Context, req list.ListRequest, stream *list.ListResultsStream) {
	if r.client == nil {
		stream.Results = list.ListResultsStreamDiagnostics(diag.Diagnostics{
			diag.NewErrorDiagnostic(
				"Unconfigured Provider",
				"The provider has not been configured yet. Re-run the command after `terraform init` completes successfully.",
			),
		})
		return
	}

	var config JSONWebTokenConfigurationListResourceModel
	diags := req.Config.Get(ctx, &config)
	if diags.HasError() {
		stream.Results = list.ListResultsStreamDiagnostics(diags)
		return
	}

	listCtx, cancel := context.WithTimeout(ctx, defaultListTimeout)
	defer cancel()

	resp, err := r.client.ListJsonWebTokenConfigurations(listCtx)
	if err != nil {
		stream.Results = list.ListResultsStreamDiagnostics(diag.Diagnostics{
			diag.NewErrorDiagnostic("Unable to list Jamf Pro JSON Web Token configurations", err.Error()),
		})
		return
	}

	items := []proclassic.JsonWebTokenConfiguration{}
	if resp != nil {
		items = resp.JsonWebTokenConfigurations
	}

	filter := filters.ClassicFilterModel{}
	if config.Filter != nil {
		filter = *config.Filter
	}
	items = filters.ApplyClassicFilter(items, filter, jsonWebTokenConfigurationListItemName)

	maxResults := req.Limit
	if maxResults <= 0 || maxResults > int64(len(items)) {
		maxResults = int64(len(items))
	}

	results := make([]list.ListResult, 0, maxResults)

	for _, item := range items {
		if int64(len(results)) >= maxResults {
			break
		}

		result := req.NewListResult(ctx)
		result.DisplayName = helpers.DerefString(item.Name)

		id := helpers.StringValueFromIntPtr(item.ID)
		result.Diagnostics.Append(helpers.SetIdentity(ctx, result.Identity, jsonWebTokenConfigurationIdentityModel{ID: id})...)
		if result.Diagnostics.HasError() {
			stream.Results = list.ListResultsStreamDiagnostics(result.Diagnostics)
			return
		}

		results = append(results, result)
	}

	tflog.Debug(ctx, "Listed Jamf Pro JSON Web Token configurations", map[string]any{
		"name_substring": filter.NameSubstring.ValueString(),
		"limit":          req.Limit,
		"returned":       len(results),
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

// jsonWebTokenConfigurationListItemName is the name accessor passed to
// filters.ApplyClassicFilter.
func jsonWebTokenConfigurationListItemName(item proclassic.JsonWebTokenConfiguration) string {
	return helpers.DerefString(item.Name)
}
