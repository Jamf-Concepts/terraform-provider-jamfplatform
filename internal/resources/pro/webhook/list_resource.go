// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package webhook

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
// /webhooks endpoint.
const defaultListTimeout = 90 * time.Second

var (
	_ list.ListResource              = &WebhookListResource{}
	_ list.ListResourceWithConfigure = &WebhookListResource{}
)

// NewWebhookListResource returns a list resource for Jamf Pro webhook queries.
func NewWebhookListResource() list.ListResource {
	return &WebhookListResource{}
}

// WebhookListResource implements Terraform query list support for Jamf Pro
// webhooks. Classic /webhooks has no RSQL — the optional `filter` block is
// applied client-side via filters.ApplyClassicFilter. List items carry only
// id + name on the wire, so when IncludeResource is requested (config
// generation) each webhook is fetched individually and hydrated through the
// shared Read state-builder — matching the resource's import fidelity.
type WebhookListResource struct {
	client *proclassic.Client
}

// Metadata sets the list resource type name.
func (r *WebhookListResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pro_webhook"
}

// Configure wires the Jamf ProClassic client into the list resource.
func (r *WebhookListResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	client, diags := providerdata.ConfigureProClassic(ctx, req.ProviderData, minJamfProVersion, "jamfplatform_pro_webhook")
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = client
}

// ListResourceConfigSchema describes the supported list filters.
func (r *WebhookListResource) ListResourceConfigSchema(ctx context.Context, req list.ListResourceSchemaRequest, resp *list.ListResourceSchemaResponse) {
	resp.Schema = listschema.Schema{
		Description: "Lists Jamf Pro webhooks. Supply an optional case-insensitive `name_substring` filter applied locally after the full list is fetched. List entries surface as identity-only (id and display name); full detail requires a per-record read.",
		Attributes: map[string]listschema.Attribute{
			"filter": filters.ClassicListFilterAttribute(),
		},
	}
}

// List executes the query and streams record identities back to Terraform.
func (r *WebhookListResource) List(ctx context.Context, req list.ListRequest, stream *list.ListResultsStream) {
	if r.client == nil {
		stream.Results = list.ListResultsStreamDiagnostics(diag.Diagnostics{
			diag.NewErrorDiagnostic(
				"Unconfigured Provider",
				"The provider has not been configured yet. Re-run the command after `terraform init` completes successfully.",
			),
		})
		return
	}

	var config WebhookListResourceModel
	diags := req.Config.Get(ctx, &config)
	if diags.HasError() {
		stream.Results = list.ListResultsStreamDiagnostics(diags)
		return
	}

	listCtx, cancel := context.WithTimeout(ctx, defaultListTimeout)
	defer cancel()

	resp, err := r.client.ListWebhooks(listCtx)
	if err != nil {
		stream.Results = list.ListResultsStreamDiagnostics(diag.Diagnostics{
			diag.NewErrorDiagnostic("Unable to list Jamf Pro webhooks", err.Error()),
		})
		return
	}

	items := []proclassic.IDName{}
	if resp != nil {
		items = resp.Webhooks
	}

	filter := filters.ClassicFilterModel{}
	if config.Filter != nil {
		filter = *config.Filter
	}
	items = filters.ApplyClassicFilter(items, filter, webhookListItemName)

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
		result.Diagnostics.Append(helpers.SetIdentity(ctx, result.Identity, webhookIdentityModel{ID: id})...)
		if result.Diagnostics.HasError() {
			stream.Results = list.ListResultsStreamDiagnostics(result.Diagnostics)
			return
		}

		if req.IncludeResource {
			got, err := r.client.GetWebhookByID(listCtx, id.ValueString())
			if err != nil {
				result.Diagnostics.AddError("Unable to read webhook", err.Error())
				stream.Results = list.ListResultsStreamDiagnostics(result.Diagnostics)
				return
			}
			state := WebhookResourceModel{
				ID:       id,
				Timeouts: helpers.NewResourceTimeoutsNullValue(webhookTimeoutAttributeTypes),
			}
			result.Diagnostics.Append(assignWebhookResourceModel(listCtx, &state, got)...)
			result.Diagnostics.Append(result.Resource.Set(ctx, &state)...)
			if result.Diagnostics.HasError() {
				stream.Results = list.ListResultsStreamDiagnostics(result.Diagnostics)
				return
			}
		}

		results = append(results, result)
	}

	tflog.Debug(ctx, "Listed Jamf Pro webhooks", map[string]any{
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

// webhookListItemName is the name accessor passed to filters.ApplyClassicFilter.
func webhookListItemName(item proclassic.IDName) string {
	return helpers.DerefString(item.Name)
}
