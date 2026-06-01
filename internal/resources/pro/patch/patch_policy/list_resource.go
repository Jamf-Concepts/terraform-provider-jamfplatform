// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package patch_policy

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
// /patchpolicies endpoint. The list resource schema does not expose a
// user-overridable timeout, so this is a fixed safety bound.
const defaultListTimeout = 90 * time.Second

var _ list.ListResource = &PatchPolicyListResource{}
var _ list.ListResourceWithConfigure = &PatchPolicyListResource{}

// NewPatchPolicyListResource returns a list resource for Jamf Pro patch policy
// queries.
func NewPatchPolicyListResource() list.ListResource {
	return &PatchPolicyListResource{}
}

// PatchPolicyListResource implements Terraform query list support. Classic
// /patchpolicies accepts no query parameters, so the optional `filter` block is
// applied client-side via filters.ApplyClassicFilter after the full list is
// fetched. Unlike patch software titles, the list response carries a display
// name, so the filter matches the policy name.
type PatchPolicyListResource struct {
	client *proclassic.Client
}

// Metadata sets the list resource type name.
func (r *PatchPolicyListResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pro_patch_policy"
}

// Configure wires the Jamf ProClassic client into the list resource via the shared
// providerdata.ConfigureProClassic helper.
func (r *PatchPolicyListResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	client, diags := providerdata.ConfigureProClassic(ctx, req.ProviderData, minJamfProVersion, "jamfplatform_pro_patch_policy")
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = client
}

// ListResourceConfigSchema describes the supported list filters.
func (r *PatchPolicyListResource) ListResourceConfigSchema(ctx context.Context, req list.ListResourceSchemaRequest, resp *list.ListResourceSchemaResponse) {
	resp.Schema = listschema.Schema{
		Description: "Lists Jamf Pro patch policies. Supply an optional case-insensitive `name_substring` filter; filtering is applied client-side after the full list is fetched against the policy display name.",
		Attributes: map[string]listschema.Attribute{
			"filter": filters.ClassicListFilterAttribute(),
		},
	}
}

// List executes the query and streams patch policy identities back to Terraform.
//
// The classic /patchpolicies list endpoint returns only id+name per item. When
// include_resource is requested we therefore hydrate only id and name; the
// remaining attributes are set null rather than issuing a per-item GET against
// this concurrency-sensitive classic endpoint. Consumers needing full detail
// should use the singular data source or the managed resource.
func (r *PatchPolicyListResource) List(ctx context.Context, req list.ListRequest, stream *list.ListResultsStream) {
	if r.client == nil {
		stream.Results = list.ListResultsStreamDiagnostics(diag.Diagnostics{
			diag.NewErrorDiagnostic(
				"Unconfigured Provider",
				"The provider has not been configured yet. Re-run the command after `terraform init` completes successfully.",
			),
		})
		return
	}

	var config PatchPolicyListResourceModel
	diags := req.Config.Get(ctx, &config)
	if diags.HasError() {
		stream.Results = list.ListResultsStreamDiagnostics(diags)
		return
	}

	listCtx, cancel := context.WithTimeout(ctx, defaultListTimeout)
	defer cancel()

	// NOTE: only the *list* endpoints carry the SDK Deprecated marker (the spec
	// points at /v2/patch-policies for listing); the by-ID CRUD funcs this
	// resource otherwise uses are not deprecated. The classic list remains the
	// only functional list surface, so it is intentionally used here.
	resp, err := r.client.ListPatchPolicies(listCtx) //nolint:staticcheck // SA1019: classic /patchpolicies list intentionally used; only list endpoints are spec-deprecated
	if err != nil {
		stream.Results = list.ListResultsStreamDiagnostics(diag.Diagnostics{
			diag.NewErrorDiagnostic("Unable to list Jamf Pro patch policies", err.Error()),
		})
		return
	}

	items := []proclassic.IDName{}
	if resp != nil {
		items = resp.PatchPolicies
	}

	filter := filters.ClassicFilterModel{}
	if config.Filter != nil {
		filter = *config.Filter
	}
	items = filters.ApplyClassicFilter(items, filter, patchPolicyName)

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
		result.Diagnostics.Append(helpers.SetIdentity(ctx, result.Identity, patchPolicyIdentityModel{ID: id})...)
		if result.Diagnostics.HasError() {
			stream.Results = list.ListResultsStreamDiagnostics(result.Diagnostics)
			return
		}

		if req.IncludeResource {
			// The list endpoint exposes only id+name; the remaining attributes
			// are left null (see method doc).
			objType := types.ObjectType{AttrTypes: killAppAttrTypes}
			state := PatchPolicyResourceModel{
				ID:                           id,
				SoftwareTitleConfigurationID: types.StringNull(),
				Name:                         helpers.StringPointerValueOrNull(s.Name),
				Enabled:                      types.BoolNull(),
				TargetVersion:                types.StringNull(),
				DistributionMethod:           types.StringNull(),
				AllowDowngrade:               types.BoolNull(),
				PatchUnknown:                 types.BoolNull(),
				ReleaseDate:                  types.Int64Null(),
				IncrementalUpdate:            types.BoolNull(),
				Reboot:                       types.BoolNull(),
				MinimumOS:                    types.StringNull(),
				KillApps:                     types.ListNull(objType),
				Timeouts:                     helpers.NewResourceTimeoutsNullValue(patchPolicyTimeoutAttributeTypes),
			}
			result.Diagnostics.Append(result.Resource.Set(ctx, &state)...)
			if result.Diagnostics.HasError() {
				stream.Results = list.ListResultsStreamDiagnostics(result.Diagnostics)
				return
			}
		}

		results = append(results, result)
	}

	tflog.Debug(ctx, "Listed Jamf Pro patch policies", map[string]any{
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

// patchPolicyName is the name accessor passed to filters.ApplyClassicFilter.
func patchPolicyName(s proclassic.IDName) string {
	return helpers.DerefString(s.Name)
}
