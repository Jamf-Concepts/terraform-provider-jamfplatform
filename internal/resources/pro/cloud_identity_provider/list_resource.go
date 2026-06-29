// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

// SDK endpoints used:
//
//	pro.ListCloudIdpV1
//
// Status: current. Last reviewed 2026-05-30.
package cloud_identity_provider

import (
	"context"
	"time"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/pro"
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

// defaultListTimeout caps how long the list operation will wait on the
// Pro Cloud Identity Provider registry (/v1/cloud-idp) endpoint. The list resource schema does not
// expose a user-overridable timeout, so this is a fixed safety bound.
const defaultListTimeout = 90 * time.Second

var (
	_ list.ListResource              = &CloudIdentityProviderListResource{}
	_ list.ListResourceWithConfigure = &CloudIdentityProviderListResource{}
)

// CloudIdentityProviderListResource implements Terraform query list support for
// Jamf Pro Cloud Identity Providers. The Pro Cloud Identity Provider registry (/v1/cloud-idp) list
// endpoint has no filter or RSQL parameters, so the optional `filter` block is
// applied client-side via filters.ApplyClassicFilter after the full list is
// fetched. The list endpoint returns the CloudIDPCommonResponse shape (all five
// fields including enabled and provider_description), so no follow-up GET is
// needed — the full summary is available in a single round trip even when
// include_resource = true.
type CloudIdentityProviderListResource struct {
	client *pro.Client
}

// NewCloudIdentityProviderListResource returns a list resource for Jamf Pro
// Cloud Identity Provider queries.
func NewCloudIdentityProviderListResource() list.ListResource {
	return &CloudIdentityProviderListResource{}
}

// Metadata sets the list resource type name.
func (r *CloudIdentityProviderListResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pro_cloud_identity_provider"
}

// Configure wires the Jamf Pro client into the list resource via the shared
// providerdata.ConfigurePro helper.
func (r *CloudIdentityProviderListResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	client, diags := providerdata.ConfigurePro(ctx, req.ProviderData, minJamfProVersion, "jamfplatform_pro_cloud_identity_provider")
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = client
}

// ListResourceConfigSchema describes the supported list filters.
func (r *CloudIdentityProviderListResource) ListResourceConfigSchema(ctx context.Context, req list.ListResourceSchemaRequest, resp *list.ListResourceSchemaResponse) {
	resp.Schema = listschema.Schema{
		Description: "Lists Jamf Pro Cloud Identity Providers (Google Secure LDAP and Microsoft Entra ID). " +
			"Supply an optional case-insensitive `name_substring` filter applied client-side after the full list is fetched. " +
			"The list response includes all summary fields (id, display_name, provider_name, enabled, provider_description); " +
			"setting `include_resource = true` populates the managed-resource state from those same fields without an extra round trip." + listResourcePrivileges,
		Attributes: map[string]listschema.Attribute{
			"filter": filters.ClassicListFilterAttribute(),
		},
	}
}

// List executes the query and streams Cloud Identity Provider identities back
// to Terraform.
func (r *CloudIdentityProviderListResource) List(ctx context.Context, req list.ListRequest, stream *list.ListResultsStream) {
	if r.client == nil {
		stream.Results = list.ListResultsStreamDiagnostics(diag.Diagnostics{
			diag.NewErrorDiagnostic(
				"Unconfigured Provider",
				"The provider has not been configured yet. Re-run the command after `terraform init` completes successfully.",
			),
		})
		return
	}

	var config CloudIdentityProviderListResourceModel
	diags := req.Config.Get(ctx, &config)
	if diags.HasError() {
		stream.Results = list.ListResultsStreamDiagnostics(diags)
		return
	}

	listCtx, cancel := context.WithTimeout(ctx, defaultListTimeout)
	defer cancel()

	all, err := r.client.ListCloudIdpV1(listCtx, nil)
	if err != nil {
		stream.Results = list.ListResultsStreamDiagnostics(diag.Diagnostics{
			diag.NewErrorDiagnostic("Unable to list Jamf Pro Cloud Identity Providers", err.Error()),
		})
		return
	}

	filter := filters.ClassicFilterModel{}
	if config.Filter != nil {
		filter = *config.Filter
	}
	all = filters.ApplyClassicFilter(all, filter, cloudIdentityProviderListItemName)

	maxResults := req.Limit
	if maxResults <= 0 || maxResults > int64(len(all)) {
		maxResults = int64(len(all))
	}

	results := make([]list.ListResult, 0, maxResults)

	for i := range all {
		if int64(len(results)) >= maxResults {
			break
		}
		item := all[i]

		result := req.NewListResult(ctx)
		result.DisplayName = item.DisplayName

		result.Diagnostics.Append(helpers.SetIdentity(ctx, result.Identity, cloudIdentityProviderIdentityModel{
			ID: types.StringValue(item.ID),
		})...)
		if result.Diagnostics.HasError() {
			stream.Results = list.ListResultsStreamDiagnostics(result.Diagnostics)
			return
		}

		if req.IncludeResource {
			// The list endpoint returns the full CloudIDPCommonResponse shape
			// (all five summary fields) so we can populate the resource state
			// directly without a follow-up GetCloudIdpV1 call. The provider-
			// specific google/azure blocks are left nil; Terraform refreshes
			// the full state on the next plan/apply.
			state := CloudIdentityProviderResourceModel{
				ID:           types.StringValue(item.ID),
				DisplayName:  types.StringValue(item.DisplayName),
				ProviderName: types.StringValue(providerNameFromWire(item.ProviderName)),
				Google:       nil,
				Azure:        nil,
				Timeouts:     helpers.NewResourceTimeoutsNullValue(cloudIdentityProviderTimeoutAttributeTypes),
			}
			result.Diagnostics.Append(result.Resource.Set(ctx, &state)...)
			if result.Diagnostics.HasError() {
				stream.Results = list.ListResultsStreamDiagnostics(result.Diagnostics)
				return
			}
		}

		results = append(results, result)
	}

	tflog.Debug(ctx, "Listed Jamf Pro Cloud Identity Providers", map[string]any{
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

// cloudIdentityProviderListItemName is the name accessor passed to
// filters.ApplyClassicFilter. DisplayName is a plain string field.
func cloudIdentityProviderListItemName(item pro.CloudIDPCommonResponse) string {
	return item.DisplayName
}
