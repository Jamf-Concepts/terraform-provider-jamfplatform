// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package advanced_computer_search

import (
	"context"
	"time"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/proclassic"
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

// defaultListTimeout caps how long the list operation waits on the classic
// /advancedcomputersearches endpoint.
const defaultListTimeout = 90 * time.Second

var _ list.ListResource = &AdvancedComputerSearchListResource{}
var _ list.ListResourceWithConfigure = &AdvancedComputerSearchListResource{}

// NewAdvancedComputerSearchListResource returns a list resource for advanced
// computer search queries.
func NewAdvancedComputerSearchListResource() list.ListResource {
	return &AdvancedComputerSearchListResource{}
}

// AdvancedComputerSearchListResource implements Terraform query list support.
// Classic /advancedcomputersearches has no RSQL — the optional `filter` block is
// applied client-side via filters.ApplyClassicFilter after the full list is
// fetched. List items carry only id and name on the wire; every other resource
// attribute is set to null on list results.
type AdvancedComputerSearchListResource struct {
	client *proclassic.Client
}

// AdvancedComputerSearchListResourceModel is the config model for list queries.
type AdvancedComputerSearchListResourceModel struct {
	Filter *filters.ClassicFilterModel `tfsdk:"filter"`
}

// Metadata sets the list resource type name.
func (r *AdvancedComputerSearchListResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pro_advanced_computer_search"
}

// Configure wires the Jamf ProClassic client into the list resource.
func (r *AdvancedComputerSearchListResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	client, diags := providerdata.ConfigureProClassic(ctx, req.ProviderData, minJamfProVersion, "jamfplatform_pro_advanced_computer_search")
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = client
}

// ListResourceConfigSchema describes the supported list filters.
func (r *AdvancedComputerSearchListResource) ListResourceConfigSchema(ctx context.Context, req list.ListResourceSchemaRequest, resp *list.ListResourceSchemaResponse) {
	resp.Schema = listschema.Schema{
		Description: "Lists Jamf Pro advanced computer searches. Supply an optional case-insensitive `name_substring` filter applied client-side after the full list is fetched.",
		Attributes: map[string]listschema.Attribute{
			"filter": filters.ClassicListFilterAttribute(),
		},
	}
}

// List executes the query and streams advanced computer search identities back
// to Terraform.
func (r *AdvancedComputerSearchListResource) List(ctx context.Context, req list.ListRequest, stream *list.ListResultsStream) {
	if r.client == nil {
		stream.Results = list.ListResultsStreamDiagnostics(diag.Diagnostics{
			diag.NewErrorDiagnostic(
				"Unconfigured Provider",
				"The provider has not been configured yet. Re-run the command after `terraform init` completes successfully.",
			),
		})
		return
	}

	var config AdvancedComputerSearchListResourceModel
	diags := req.Config.Get(ctx, &config)
	if diags.HasError() {
		stream.Results = list.ListResultsStreamDiagnostics(diags)
		return
	}

	listCtx, cancel := context.WithTimeout(ctx, defaultListTimeout)
	defer cancel()

	resp, err := r.client.ListAdvancedComputerSearches(listCtx)
	if err != nil {
		stream.Results = list.ListResultsStreamDiagnostics(diag.Diagnostics{
			diag.NewErrorDiagnostic("Unable to list Jamf Pro advanced computer searches", err.Error()),
		})
		return
	}

	items := []proclassic.IDName{}
	if resp != nil {
		items = resp.AdvancedComputerSearches
	}

	filter := filters.ClassicFilterModel{}
	if config.Filter != nil {
		filter = *config.Filter
	}
	items = filters.ApplyClassicFilter(items, filter, advancedComputerSearchListItemName)

	maxResults := req.Limit
	if maxResults <= 0 || maxResults > int64(len(items)) {
		maxResults = int64(len(items))
	}

	results := make([]list.ListResult, 0, maxResults)

	for _, s := range items {
		if int64(len(results)) >= maxResults {
			break
		}

		result := req.NewListResult(ctx)
		result.DisplayName = helpers.DerefString(s.Name)

		id := helpers.StringValueFromIntPtr(s.ID)
		result.Diagnostics.Append(helpers.SetIdentity(ctx, result.Identity, advancedComputerSearchIdentityModel{ID: id})...)
		if result.Diagnostics.HasError() {
			stream.Results = list.ListResultsStreamDiagnostics(result.Diagnostics)
			return
		}

		if req.IncludeResource {
			// List response carries id and name only. Every other Optional/Computed
			// attribute is null on list results.
			state := AdvancedComputerSearchResourceModel{
				ID:            id,
				Name:          helpers.StringPointerValueOrNull(s.Name),
				SiteID:        types.StringNull(),
				SiteName:      types.StringNull(),
				Criteria:      nil,
				DisplayFields: types.SetNull(types.StringType),
				Timeouts:      helpers.NewResourceTimeoutsNullValue(advancedComputerSearchTimeoutAttributeTypes),
			}
			result.Diagnostics.Append(result.Resource.Set(ctx, &state)...)
			if result.Diagnostics.HasError() {
				stream.Results = list.ListResultsStreamDiagnostics(result.Diagnostics)
				return
			}
		}

		results = append(results, result)
	}

	tflog.Debug(ctx, "Listed Jamf Pro advanced computer searches", map[string]any{
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

// advancedComputerSearchListItemName is the name accessor passed to
// filters.ApplyClassicFilter.
func advancedComputerSearchListItemName(s proclassic.IDName) string {
	return helpers.DerefString(s.Name)
}
