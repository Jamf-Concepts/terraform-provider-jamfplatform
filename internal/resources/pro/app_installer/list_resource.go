// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package app_installer

import (
	"context"
	"time"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/pro"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/list"
	listschema "github.com/hashicorp/terraform-plugin-framework/list/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/filters"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/providerdata"
)

// defaultListTimeout caps how long the list operation waits on the deployments
// endpoint.
const defaultListTimeout = 90 * time.Second

var _ list.ListResource = &AppInstallerListResource{}
var _ list.ListResourceWithConfigure = &AppInstallerListResource{}

// NewAppInstallerListResource returns a list resource for App Installer
// deployment queries.
func NewAppInstallerListResource() list.ListResource {
	return &AppInstallerListResource{}
}

// AppInstallerListResource implements Terraform query list support for App
// Installer deployments. The deployments endpoint has no server-side filter, so
// the optional `filter` block is applied client-side as a case-insensitive name
// substring. List items carry identity (id) + display name only, so when
// IncludeResource is requested (config generation) each deployment is fetched
// individually and hydrated through the shared Read state-builder — matching
// the resource's import fidelity.
type AppInstallerListResource struct {
	client *pro.Client
}

// Metadata sets the list resource type name.
func (r *AppInstallerListResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pro_app_installer"
}

// Configure wires the Jamf Pro client into the list resource.
func (r *AppInstallerListResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	client, diags := providerdata.ConfigurePro(ctx, req.ProviderData, minJamfProVersion, "jamfplatform_pro_app_installer")
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = client
}

// ListResourceConfigSchema describes the supported list filters.
func (r *AppInstallerListResource) ListResourceConfigSchema(ctx context.Context, req list.ListResourceSchemaRequest, resp *list.ListResourceSchemaResponse) {
	resp.Schema = listschema.Schema{
		Description: "Lists App Installer deployments. Supply an optional case-insensitive `name_substring` filter applied locally after the full list is fetched. List entries surface as identity-only (id and display name); full detail requires a per-deployment read.",
		Attributes: map[string]listschema.Attribute{
			"filter": filters.ClassicListFilterAttribute(),
		},
	}
}

// List executes the query and streams deployment identities back to Terraform.
func (r *AppInstallerListResource) List(ctx context.Context, req list.ListRequest, stream *list.ListResultsStream) {
	if r.client == nil {
		stream.Results = list.ListResultsStreamDiagnostics(diag.Diagnostics{
			diag.NewErrorDiagnostic(
				"Unconfigured Provider",
				"The provider has not been configured yet. Re-run the command after `terraform init` completes successfully.",
			),
		})
		return
	}

	var config AppInstallerListResourceModel
	diags := req.Config.Get(ctx, &config)
	if diags.HasError() {
		stream.Results = list.ListResultsStreamDiagnostics(diags)
		return
	}

	listCtx, cancel := context.WithTimeout(ctx, defaultListTimeout)
	defer cancel()

	entries, err := r.client.ListAppInstallerDeploymentsV1(listCtx)
	if err != nil {
		stream.Results = list.ListResultsStreamDiagnostics(diag.Diagnostics{
			diag.NewErrorDiagnostic("Unable to list App Installer deployments", err.Error()),
		})
		return
	}

	filter := filters.ClassicFilterModel{}
	if config.Filter != nil {
		filter = *config.Filter
	}
	entries = filters.ApplyClassicFilter(entries, filter, appInstallerListItemName)

	maxResults := req.Limit
	if maxResults <= 0 || maxResults > int64(len(entries)) {
		maxResults = int64(len(entries))
	}

	results := make([]list.ListResult, 0, maxResults)

	for _, e := range entries {
		if int64(len(results)) >= maxResults {
			break
		}

		result := req.NewListResult(ctx)
		result.DisplayName = e.Name

		result.Diagnostics.Append(helpers.SetIdentity(ctx, result.Identity, appInstallerIdentityModel{ID: helpers.StringPointerValueOrNull(&e.ID)})...)
		if result.Diagnostics.HasError() {
			stream.Results = list.ListResultsStreamDiagnostics(result.Diagnostics)
			return
		}

		if req.IncludeResource {
			got, err := r.client.GetAppInstallerDeploymentV1(listCtx, e.ID)
			if err != nil {
				result.Diagnostics.AddError("Unable to read app installer deployment", err.Error())
				stream.Results = list.ListResultsStreamDiagnostics(result.Diagnostics)
				return
			}
			state := AppInstallerResourceModel{
				ID:       helpers.StringPointerValueOrNull(&e.ID),
				Timeouts: helpers.NewResourceTimeoutsNullValue(appInstallerTimeoutAttributeTypes),
			}
			assignAppInstallerResourceModel(&state, got)
			result.Diagnostics.Append(result.Resource.Set(ctx, &state)...)
			if result.Diagnostics.HasError() {
				stream.Results = list.ListResultsStreamDiagnostics(result.Diagnostics)
				return
			}
		}

		results = append(results, result)
	}

	tflog.Debug(ctx, "Listed App Installer deployments", map[string]any{
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

// appInstallerListItemName is the name accessor passed to filters.ApplyClassicFilter.
func appInstallerListItemName(e pro.AppInstallerDeploymentListEntry) string {
	return e.Name
}
