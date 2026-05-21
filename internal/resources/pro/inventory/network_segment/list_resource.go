// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package network_segment

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

// defaultListTimeout caps how long the list operation will wait on the classic
// /networksegments endpoint. The list resource schema does not expose a user-
// overridable timeout, so this is a fixed safety bound.
const defaultListTimeout = 90 * time.Second

var _ list.ListResource = &NetworkSegmentListResource{}
var _ list.ListResourceWithConfigure = &NetworkSegmentListResource{}

// NewNetworkSegmentListResource returns a list resource for Jamf Pro network segment queries.
func NewNetworkSegmentListResource() list.ListResource {
	return &NetworkSegmentListResource{}
}

// NetworkSegmentListResource implements Terraform query list support for Jamf Pro
// network segments. Classic /networksegments accepts no query parameters, so the
// optional `filter` block is applied client-side via filters.ApplyClassicFilter
// after the full list is fetched. The list-item type carries only id, name,
// starting_address, and ending_address — every other resource attribute is set to
// null on list results.
type NetworkSegmentListResource struct {
	client *proclassic.Client
}

// Metadata sets the list resource type name.
func (r *NetworkSegmentListResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pro_network_segment"
}

// Configure wires the Jamf ProClassic client into the list resource via the shared
// providerdata.ConfigureProClassic helper.
func (r *NetworkSegmentListResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	client, diags := providerdata.ConfigureProClassic(ctx, req.ProviderData, minJamfProVersion, "jamfplatform_pro_network_segment")
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = client
}

// ListResourceConfigSchema describes the supported list filters.
func (r *NetworkSegmentListResource) ListResourceConfigSchema(ctx context.Context, req list.ListResourceSchemaRequest, resp *list.ListResourceSchemaResponse) {
	resp.Schema = listschema.Schema{
		Description: "Lists Jamf Pro network segments. Classic has no RSQL — supply an optional case-insensitive `name_substring` filter applied client-side after the full list is fetched.",
		Attributes: map[string]listschema.Attribute{
			"filter": filters.ClassicListFilterAttribute(),
		},
	}
}

// List executes the query and streams network segment identities back to Terraform.
func (r *NetworkSegmentListResource) List(ctx context.Context, req list.ListRequest, stream *list.ListResultsStream) {
	if r.client == nil {
		stream.Results = list.ListResultsStreamDiagnostics(diag.Diagnostics{
			diag.NewErrorDiagnostic(
				"Unconfigured Provider",
				"The provider has not been configured yet. Re-run the command after `terraform init` completes successfully.",
			),
		})
		return
	}

	var config NetworkSegmentListResourceModel
	diags := req.Config.Get(ctx, &config)
	if diags.HasError() {
		stream.Results = list.ListResultsStreamDiagnostics(diags)
		return
	}

	listCtx, cancel := context.WithTimeout(ctx, defaultListTimeout)
	defer cancel()

	resp, err := r.client.ListNetworkSegments(listCtx)
	if err != nil {
		stream.Results = list.ListResultsStreamDiagnostics(diag.Diagnostics{
			diag.NewErrorDiagnostic("Unable to list Jamf Pro network segments", err.Error()),
		})
		return
	}

	items := []proclassic.NetworkSegmentsItemNetworkSegment{}
	if resp != nil {
		items = resp.NetworkSegments
	}

	filter := filters.ClassicFilterModel{}
	if config.Filter != nil {
		filter = *config.Filter
	}
	items = filters.ApplyClassicFilter(items, filter, networkSegmentListItemName)

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
		result.DisplayName = derefString(s.Name)

		id := helpers.StringValueFromIntPtr(s.ID)
		result.Diagnostics.Append(helpers.SetIdentity(ctx, result.Identity, networkSegmentIdentityModel{ID: id})...)
		if result.Diagnostics.HasError() {
			stream.Results = list.ListResultsStreamDiagnostics(result.Diagnostics)
			return
		}

		if req.IncludeResource {
			// List endpoint returns only id, name, starting_address, ending_address.
			// Every other Optional/Computed attribute is null on a list result.
			state := NetworkSegmentResourceModel{
				ID:                  id,
				Name:                helpers.StringPointerValueOrNull(s.Name),
				StartingAddress:     helpers.StringPointerValueOrNull(s.StartingAddress),
				EndingAddress:       helpers.StringPointerValueOrNull(s.EndingAddress),
				Building:            types.StringNull(),
				Department:          types.StringNull(),
				OverrideBuildings:   types.BoolNull(),
				OverrideDepartments: types.BoolNull(),
				DistributionPoint:   types.StringNull(),
				DistributionServer:  types.StringNull(),
				SwuServer:           types.StringNull(),
				URL:                 types.StringNull(),
				Timeouts:            helpers.NewResourceTimeoutsNullValue(networkSegmentTimeoutAttributeTypes),
			}
			result.Diagnostics.Append(result.Resource.Set(ctx, &state)...)
			if result.Diagnostics.HasError() {
				stream.Results = list.ListResultsStreamDiagnostics(result.Diagnostics)
				return
			}
		}

		results = append(results, result)
	}

	tflog.Debug(ctx, "Listed Jamf Pro network segments", map[string]any{
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

// networkSegmentListItemName is the name accessor passed to filters.ApplyClassicFilter.
func networkSegmentListItemName(s proclassic.NetworkSegmentsItemNetworkSegment) string {
	return derefString(s.Name)
}

// derefString returns the underlying string for a non-nil *string, or "" for nil.
func derefString(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
