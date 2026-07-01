// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package patch_software_title

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
// /patchsoftwaretitles endpoint. The list resource schema does not expose a
// user-overridable timeout, so this is a fixed safety bound.
const defaultListTimeout = 90 * time.Second

var _ list.ListResource = &PatchSoftwareTitleListResource{}
var _ list.ListResourceWithConfigure = &PatchSoftwareTitleListResource{}

// NewPatchSoftwareTitleListResource returns a list resource for Jamf Pro patch
// software title queries.
func NewPatchSoftwareTitleListResource() list.ListResource {
	return &PatchSoftwareTitleListResource{}
}

// PatchSoftwareTitleListResource implements Terraform query list support. Classic
// /patchsoftwaretitles accepts no query parameters, so the optional `filter`
// block is applied client-side via filters.ApplyClassicFilter after the full
// list is fetched.
type PatchSoftwareTitleListResource struct {
	client *proclassic.Client
}

// Metadata sets the list resource type name.
func (r *PatchSoftwareTitleListResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pro_patch_software_title"
}

// Configure wires the Jamf ProClassic client into the list resource via the shared
// providerdata.ConfigureProClassic helper.
func (r *PatchSoftwareTitleListResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	client, diags := providerdata.ConfigureProClassic(ctx, req.ProviderData, minJamfProVersion, "jamfplatform_pro_patch_software_title")
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = client
}

// ListResourceConfigSchema describes the supported list filters.
func (r *PatchSoftwareTitleListResource) ListResourceConfigSchema(ctx context.Context, req list.ListResourceSchemaRequest, resp *list.ListResourceSchemaResponse) {
	resp.Schema = listschema.Schema{
		Description: "Lists Jamf Pro patch software titles. Supply an optional case-insensitive `name_substring` filter; filtering is applied client-side after the full list is fetched, matching each title's display name." +
			listResourcePrivileges,
		Attributes: map[string]listschema.Attribute{
			"filter": filters.ClassicListFilterAttribute(),
		},
	}
}

// List executes the query and streams patch software title identities back to
// Terraform.
//
// The classic /patchsoftwaretitles list endpoint returns only id+name+name_id+
// source_id per item. When include_resource is requested we therefore hydrate
// only those four attributes; the remaining attributes are set null rather than
// issuing a per-item GET against this concurrency-sensitive classic endpoint. A
// list preview does not require full hydration — consumers needing the full
// object should use the singular data source or the managed resource.
func (r *PatchSoftwareTitleListResource) List(ctx context.Context, req list.ListRequest, stream *list.ListResultsStream) {
	if r.client == nil {
		stream.Results = list.ListResultsStreamDiagnostics(diag.Diagnostics{
			diag.NewErrorDiagnostic(
				"Unconfigured Provider",
				"The provider has not been configured yet. Re-run the command after `terraform init` completes successfully.",
			),
		})
		return
	}

	var config PatchSoftwareTitleListResourceModel
	diags := req.Config.Get(ctx, &config)
	if diags.HasError() {
		stream.Results = list.ListResultsStreamDiagnostics(diags)
		return
	}

	listCtx, cancel := context.WithTimeout(ctx, defaultListTimeout)
	defer cancel()

	resp, err := r.client.ListPatchSoftwareTitles(listCtx) //nolint:staticcheck // SA1019: classic /patchsoftwaretitles intentionally used; v2 create unusable — see crud.go header note
	if err != nil {
		stream.Results = list.ListResultsStreamDiagnostics(diag.Diagnostics{
			diag.NewErrorDiagnostic("Unable to list Jamf Pro patch software titles", err.Error()),
		})
		return
	}

	items := []proclassic.PatchSoftwareTitlesItemPatchSoftwareTitle{}
	if resp != nil {
		items = resp.PatchSoftwareTitles
	}

	filter := filters.ClassicFilterModel{}
	if config.Filter != nil {
		filter = *config.Filter
	}
	items = filters.ApplyClassicFilter(items, filter, patchSoftwareTitleDisplayName)

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
		result.Diagnostics.Append(helpers.SetIdentity(ctx, result.Identity, patchSoftwareTitleIdentityModel{ID: id})...)
		if result.Diagnostics.HasError() {
			stream.Results = list.ListResultsStreamDiagnostics(result.Diagnostics)
			return
		}

		if req.IncludeResource {
			// The list endpoint exposes only id+name+name_id+source_id; the
			// remaining attributes are left null (see method doc).
			state := PatchSoftwareTitleResourceModel{
				ID:                        id,
				Name:                      helpers.StringPointerValueOrNull(s.Name),
				NameID:                    helpers.StringPointerValueOrNull(s.NameID),
				SourceID:                  int64PointerValueOrNull(s.SourceID),
				CategoryID:                types.StringNull(),
				CategoryName:              types.StringNull(),
				SiteID:                    types.StringNull(),
				SiteName:                  types.StringNull(),
				WebNotification:           types.BoolNull(),
				EmailNotification:         types.BoolNull(),
				VersionPackages:           types.MapNull(types.StringType),
				AvailableVersions:         types.ListNull(types.StringType),
				AcceptExtensionAttributes: types.BoolNull(),
				// Typed null (not the zero-value types.List, which is an
				// untyped/DynamicPseudoType list and fails the schema type check).
				ExtensionAttributes: types.ListNull(eaElementType),
				Timeouts:            helpers.NewResourceTimeoutsNullValue(patchSoftwareTitleTimeoutAttributeTypes),
			}
			result.Diagnostics.Append(result.Resource.Set(ctx, &state)...)
			if result.Diagnostics.HasError() {
				stream.Results = list.ListResultsStreamDiagnostics(result.Diagnostics)
				return
			}
		}

		results = append(results, result)
	}

	tflog.Debug(ctx, "Listed Jamf Pro patch software titles", map[string]any{
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

// patchSoftwareTitleDisplayName is the name accessor passed to
// filters.ApplyClassicFilter, matching each title's display name.
func patchSoftwareTitleDisplayName(s proclassic.PatchSoftwareTitlesItemPatchSoftwareTitle) string {
	return helpers.DerefString(s.Name)
}

// int64PointerValueOrNull maps an SDK *int onto a Terraform Int64, null for nil.
func int64PointerValueOrNull(p *int) types.Int64 {
	if p == nil {
		return types.Int64Null()
	}
	return types.Int64Value(int64(*p))
}
