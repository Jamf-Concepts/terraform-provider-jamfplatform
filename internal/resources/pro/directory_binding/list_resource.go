// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package directory_binding

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

// defaultListTimeout caps how long the list operation will wait on the
// classic /directorybindings endpoint. The list resource schema does not
// expose a user-overridable timeout, so this is a fixed safety bound.
const defaultListTimeout = 90 * time.Second

var (
	_ list.ListResource              = &DirectoryBindingListResource{}
	_ list.ListResourceWithConfigure = &DirectoryBindingListResource{}
)

// NewDirectoryBindingListResource returns a list resource for Jamf Pro
// directory binding queries.
func NewDirectoryBindingListResource() list.ListResource {
	return &DirectoryBindingListResource{}
}

// DirectoryBindingListResource implements Terraform query list support for
// Jamf Pro directory bindings. Classic /directorybindings accepts no query
// parameters, so the optional `filter` block is applied client-side via
// filters.ApplyClassicFilter after the full list is fetched. The list
// endpoint returns only id+name per row, so when IncludeResource=true we
// follow up with a per-item GetDirectoryBindingByID to populate the full
// record — N+1 path mirroring the printer and dock_item list resources.
// Identity-only listing stays a single round trip.
type DirectoryBindingListResource struct {
	client *proclassic.Client
}

// Metadata sets the list resource type name.
func (r *DirectoryBindingListResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pro_directory_binding"
}

// Configure wires the Jamf ProClassic client into the list resource via the
// shared providerdata.ConfigureProClassic helper.
func (r *DirectoryBindingListResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	client, diags := providerdata.ConfigureProClassic(ctx, req.ProviderData, minJamfProVersion, "jamfplatform_pro_directory_binding")
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = client
}

// ListResourceConfigSchema describes the supported list filters.
func (r *DirectoryBindingListResource) ListResourceConfigSchema(ctx context.Context, req list.ListResourceSchemaRequest, resp *list.ListResourceSchemaResponse) {
	resp.Schema = listschema.Schema{
		Description: "Lists Jamf Pro directory bindings. Supply an optional case-insensitive `name_substring` filter; filtering is applied client-side after the full list is fetched. The list returns `id` and `name` per row by default — setting `include_resource = true` fetches the full record for each item." + listResourcePrivileges,
		Attributes: map[string]listschema.Attribute{
			"filter": filters.ClassicListFilterAttribute(),
		},
	}
}

// List executes the query and streams directory binding identities back to
// Terraform.
func (r *DirectoryBindingListResource) List(ctx context.Context, req list.ListRequest, stream *list.ListResultsStream) {
	if r.client == nil {
		stream.Results = list.ListResultsStreamDiagnostics(diag.Diagnostics{
			diag.NewErrorDiagnostic(
				"Unconfigured Provider",
				"The provider has not been configured yet. Re-run the command after `terraform init` completes successfully.",
			),
		})
		return
	}

	var config DirectoryBindingListResourceModel
	diags := req.Config.Get(ctx, &config)
	if diags.HasError() {
		stream.Results = list.ListResultsStreamDiagnostics(diags)
		return
	}

	listCtx, cancel := context.WithTimeout(ctx, defaultListTimeout)
	defer cancel()

	resp, err := r.client.ListDirectoryBindings(listCtx)
	if err != nil {
		stream.Results = list.ListResultsStreamDiagnostics(diag.Diagnostics{
			diag.NewErrorDiagnostic("Unable to list Jamf Pro directory bindings", err.Error()),
		})
		return
	}

	items := []proclassic.IDName{}
	if resp != nil {
		items = resp.DirectoryBindings
	}

	filter := filters.ClassicFilterModel{}
	if config.Filter != nil {
		filter = *config.Filter
	}
	items = filters.ApplyClassicFilter(items, filter, directoryBindingListItemName)

	maxResults := req.Limit
	if maxResults <= 0 || maxResults > int64(len(items)) {
		maxResults = int64(len(items))
	}

	results := make([]list.ListResult, 0, maxResults)

	for _, b := range items {
		if int64(len(results)) >= maxResults {
			break
		}

		result := req.NewListResult(ctx)
		result.DisplayName = helpers.DerefString(b.Name)

		id := helpers.StringValueFromIntPtr(b.ID)
		result.Diagnostics.Append(helpers.SetIdentity(ctx, result.Identity, directoryBindingIdentityModel{ID: id})...)
		if result.Diagnostics.HasError() {
			stream.Results = list.ListResultsStreamDiagnostics(result.Diagnostics)
			return
		}

		if req.IncludeResource {
			// /directorybindings list response carries only id+name. The
			// resource schema is large (top-level envelope plus four
			// per-type nested blocks) so we follow up with a singular GET
			// to populate the full record rather than emitting nulls.
			full, err := r.client.GetDirectoryBindingByID(listCtx, id.ValueString())
			if err != nil {
				result.Diagnostics.AddError(
					"Unable to fetch full directory binding for list result",
					err.Error(),
				)
				stream.Results = list.ListResultsStreamDiagnostics(result.Diagnostics)
				return
			}
			state := DirectoryBindingResourceModel{
				ID:       id,
				Timeouts: helpers.NewResourceTimeoutsNullValue(directoryBindingTimeoutAttributeTypes),
			}
			result.Diagnostics.Append(assignDirectoryBindingResourceModel(&state, full)...)
			if result.Diagnostics.HasError() {
				stream.Results = list.ListResultsStreamDiagnostics(result.Diagnostics)
				return
			}
			result.Diagnostics.Append(result.Resource.Set(ctx, &state)...)
			if result.Diagnostics.HasError() {
				stream.Results = list.ListResultsStreamDiagnostics(result.Diagnostics)
				return
			}
		}

		results = append(results, result)
	}

	tflog.Debug(ctx, "Listed Jamf Pro directory bindings", map[string]any{
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

// directoryBindingListItemName is the name accessor passed to
// filters.ApplyClassicFilter.
func directoryBindingListItemName(b proclassic.IDName) string {
	return helpers.DerefString(b.Name)
}
