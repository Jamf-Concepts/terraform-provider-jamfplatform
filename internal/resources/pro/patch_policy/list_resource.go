// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package patch_policy

import (
	"context"
	"time"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/pro"
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

// defaultListTimeout caps how long the list operation will wait on the Pro v2
// patch-policies collection, which the SDK pages through in full. The list
// resource schema does not expose a user-overridable timeout, so this is a
// fixed safety bound.
const defaultListTimeout = 90 * time.Second

// defaultItemReadTimeout bounds each per-item GET issued when IncludeResource is
// requested (config generation). A single slow item is skipped from the
// generated config rather than aborting the whole type.
const defaultItemReadTimeout = 30 * time.Second

var _ list.ListResource = &PatchPolicyListResource{}
var _ list.ListResourceWithConfigure = &PatchPolicyListResource{}

// NewPatchPolicyListResource returns a list resource for Jamf Pro patch policy
// queries.
func NewPatchPolicyListResource() list.ListResource {
	return &PatchPolicyListResource{}
}

// PatchPolicyListResource implements Terraform query list support. Enumeration
// runs on the Pro v2 patch-policies collection; per-item hydration stays on the
// ProClassic by-ID read, which is the only surface carrying a policy's scope and
// user-interaction sections.
//
// The optional `filter` block is applied client-side via
// filters.ApplyClassicFilter after the full list is fetched: v2 accepts an RSQL
// query on `policyName`, but RSQL string comparison is case-sensitive and this
// resource's contract is a case-insensitive substring, so filtering server-side
// would silently narrow it.
type PatchPolicyListResource struct {
	client    *proclassic.Client
	proClient *pro.Client
}

// Metadata sets the list resource type name.
func (r *PatchPolicyListResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pro_patch_policy"
}

// Configure wires both clients this list resource spans: the Pro client for the
// v2 enumeration and the ProClassic client for the per-item read.
func (r *PatchPolicyListResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	client, diags := providerdata.ConfigureProClassic(ctx, req.ProviderData, minJamfProVersion, "jamfplatform_pro_patch_policy")
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	proClient, proDiags := providerdata.ConfigurePro(ctx, req.ProviderData, minJamfProVersion, "jamfplatform_pro_patch_policy")
	resp.Diagnostics.Append(proDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = client
	r.proClient = proClient
}

// ListResourceConfigSchema describes the supported list filters.
func (r *PatchPolicyListResource) ListResourceConfigSchema(ctx context.Context, req list.ListResourceSchemaRequest, resp *list.ListResourceSchemaResponse) {
	resp.Schema = listschema.Schema{
		Description: "Lists Jamf Pro patch policies. Supply an optional case-insensitive `name_substring` filter; filtering is applied client-side after the full list is fetched against the policy display name." + listResourcePrivileges,
		Attributes: map[string]listschema.Attribute{
			"filter": filters.ClassicListFilterAttribute(),
		},
	}
}

// List executes the query and streams patch policy identities back to Terraform.
//
// The Pro v2 collection returns a summary view per policy (id, name, target
// version, deployment method, deployment counts) and no scope or
// user-interaction sections, so when IncludeResource is requested (config
// generation) each policy is fetched individually through the ProClassic by-ID
// read and hydrated through the shared Read state-builder with
// includeUnmanaged=true, populating every wire-present section so the generated
// config is complete rather than identity-only.
func (r *PatchPolicyListResource) List(ctx context.Context, req list.ListRequest, stream *list.ListResultsStream) {
	if r.client == nil || r.proClient == nil {
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

	items, err := r.proClient.ListPatchPoliciesV2(listCtx, nil, "")
	if err != nil {
		stream.Results = list.ListResultsStreamDiagnostics(diag.Diagnostics{
			diag.NewErrorDiagnostic("Unable to list Jamf Pro patch policies", err.Error()),
		})
		return
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
		result.DisplayName = s.PolicyName

		id := types.StringValue(s.ID)
		result.Diagnostics.Append(helpers.SetIdentity(ctx, result.Identity, patchPolicyIdentityModel{ID: id})...)
		if result.Diagnostics.HasError() {
			stream.Results = list.ListResultsStreamDiagnostics(result.Diagnostics)
			return
		}

		if req.IncludeResource {
			itemCtx, cancel := context.WithTimeout(ctx, defaultItemReadTimeout)
			got, err := r.client.GetPatchPolicyByID(itemCtx, id.ValueString())
			cancel()
			if err != nil {
				tflog.Warn(ctx, "Skipping Jamf Pro patch policy from generated config after per-item read failure", map[string]any{
					"id":    id.ValueString(),
					"error": err.Error(),
				})
				continue
			}
			state := PatchPolicyResourceModel{
				ID:       id,
				Timeouts: helpers.NewResourceTimeoutsNullValue(patchPolicyTimeoutAttributeTypes),
			}
			result.Diagnostics.Append(assignPatchPolicyResourceModel(ctx, &state, got, true)...)
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
func patchPolicyName(s pro.PatchPolicyListView) string {
	return s.PolicyName
}
