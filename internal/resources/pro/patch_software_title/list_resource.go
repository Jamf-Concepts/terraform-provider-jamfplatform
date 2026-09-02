// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package patch_software_title

import (
	"context"
	"fmt"
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

// defaultListTimeout caps how long the list operation will wait on the v3
// configurations list plus the two patch-source catalogue reads. The list
// resource schema does not expose a user-overridable timeout, so this is a
// fixed safety bound.
const defaultListTimeout = 90 * time.Second

var _ list.ListResource = &PatchSoftwareTitleListResource{}
var _ list.ListResourceWithConfigure = &PatchSoftwareTitleListResource{}

// NewPatchSoftwareTitleListResource returns a list resource for Jamf Pro patch
// software title queries.
func NewPatchSoftwareTitleListResource() list.ListResource {
	return &PatchSoftwareTitleListResource{}
}

// PatchSoftwareTitleListResource implements Terraform query list support. The v3
// configurations list accepts no query parameters, so the optional `filter`
// block is applied client-side via filters.ApplyClassicFilter after the full
// list is fetched.
//
// The classic client is here only to resolve source_id: the v3 configuration
// names a title's patch source but never numbers it (see resolveSourceID).
type PatchSoftwareTitleListResource struct {
	client    *proclassic.Client
	proClient *pro.Client
}

// Metadata sets the list resource type name.
func (r *PatchSoftwareTitleListResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pro_patch_software_title"
}

// Configure wires both Jamf clients into the list resource: the Pro client for
// the v3 configurations list, and the ProClassic client for patch-source name
// resolution.
func (r *PatchSoftwareTitleListResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	client, diags := providerdata.ConfigureProClassic(ctx, req.ProviderData, minJamfProVersion, "jamfplatform_pro_patch_software_title")
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = client

	proClient, proDiags := providerdata.ConfigurePro(ctx, req.ProviderData, minJamfProVersion, "jamfplatform_pro_patch_software_title")
	resp.Diagnostics.Append(proDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.proClient = proClient
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
// The v3 configurations list returns whole configuration objects rather than
// stubs, so include_resource hydrates everything that payload carries. Two
// attributes stay null because filling them would cost a call per item:
// available_versions, which lives on the /definitions sub-resource, and
// extension_attributes, whose display names come from the /extension-attributes
// sub-resource (the configuration body carries only ids and accept flags). A
// list preview does not require full hydration — consumers needing those should
// use the singular data source or the managed resource.
//
// source_id is not on the payload either, but it resolves from the patch source
// name through two small tenant-wide catalogue reads, done once for the whole
// list rather than per item.
func (r *PatchSoftwareTitleListResource) List(ctx context.Context, req list.ListRequest, stream *list.ListResultsStream) {
	if r.client == nil || r.proClient == nil {
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

	items, err := r.proClient.ListPatchSoftwareTitleConfigurationsV3(listCtx)
	if err != nil {
		stream.Results = list.ListResultsStreamDiagnostics(diag.Diagnostics{
			diag.NewErrorDiagnostic("Unable to list Jamf Pro patch software titles", err.Error()),
		})
		return
	}

	sourceIDs, err := patchSourceIDsByName(listCtx, r.client)
	if err != nil {
		stream.Results = list.ListResultsStreamDiagnostics(diag.Diagnostics{
			diag.NewErrorDiagnostic("Unable to read Jamf Pro patch sources", err.Error()),
		})
		return
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
		result.DisplayName = s.DisplayName

		id := types.StringValue(s.ID)
		result.Diagnostics.Append(helpers.SetIdentity(ctx, result.Identity, patchSoftwareTitleIdentityModel{ID: id})...)
		if result.Diagnostics.HasError() {
			stream.Results = list.ListResultsStreamDiagnostics(result.Diagnostics)
			return
		}

		if req.IncludeResource {
			assignments, assignDiags := types.MapValueFrom(ctx, types.StringType, assignedPackagesByVersion(s.Packages))
			if assignDiags.HasError() {
				stream.Results = list.ListResultsStreamDiagnostics(assignDiags)
				return
			}

			// available_versions and extension_attributes stay null: both cost a
			// call per item (see method doc).
			state := PatchSoftwareTitleResourceModel{
				ID:                        id,
				Name:                      types.StringValue(s.DisplayName),
				NameID:                    types.StringValue(s.SoftwareTitleNameID),
				SourceID:                  sourceIDs[s.PatchSourceName],
				CategoryID:                refIDValue(s.CategoryID),
				SiteID:                    refIDValue(s.SiteID),
				WebNotification:           types.BoolValue(s.UiNotifications),
				EmailNotification:         types.BoolValue(s.EmailNotifications),
				VersionPackages:           assignments,
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
func patchSoftwareTitleDisplayName(s pro.PatchSoftwareTitleConfiguration) string {
	return s.DisplayName
}

// patchSourceIDsByName builds a patch source name → id lookup from the two
// classic source catalogues, so a list hydrates source_id for every title
// without a per-item resolve. A name present in both catalogues is omitted
// rather than guessed: the two id spaces are separate, and source_id backs a
// RequiresReplace attribute, so a wrong number is worse than none.
func patchSourceIDsByName(ctx context.Context, c *proclassic.Client) (map[string]types.Int64, error) {
	out := map[string]types.Int64{}
	if c == nil {
		return out, nil
	}

	internal, err := c.ListPatchInternalSources(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing internal patch sources: %w", err)
	}
	external, err := c.ListPatchExternalSources(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing external patch sources: %w", err)
	}

	ambiguous := map[string]bool{}
	add := func(sources []proclassic.IDName) {
		for i := range sources {
			if sources[i].Name == nil || sources[i].ID == nil {
				continue
			}
			name := *sources[i].Name
			if _, seen := out[name]; seen {
				ambiguous[name] = true
				continue
			}
			out[name] = types.Int64Value(int64(*sources[i].ID))
		}
	}
	if internal != nil {
		add(internal.PatchInternalSources)
	}
	if external != nil {
		add(external.PatchExternalSources)
	}
	for name := range ambiguous {
		delete(out, name)
	}
	return out, nil
}
