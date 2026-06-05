// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package mobile_device_provisioning_profile

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
// endpoint. The list resource schema does not expose a user-overridable timeout.
const defaultListTimeout = 90 * time.Second

var _ list.ListResource = &ProvisioningProfileListResource{}
var _ list.ListResourceWithConfigure = &ProvisioningProfileListResource{}

// NewProvisioningProfileListResource returns a list resource for provisioning profile queries.
func NewProvisioningProfileListResource() list.ListResource {
	return &ProvisioningProfileListResource{}
}

// ProvisioningProfileListResource implements Terraform query list support. The
// classic endpoint accepts no query parameters, so the optional `filter` block is
// applied client-side via filters.ApplyClassicFilter after the full list is fetched.
type ProvisioningProfileListResource struct {
	client *proclassic.Client
}

// Metadata sets the list resource type name.
func (r *ProvisioningProfileListResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pro_mobile_device_provisioning_profile"
}

// Configure wires the Jamf ProClassic client into the list resource via the shared
// providerdata.ConfigureProClassic helper.
func (r *ProvisioningProfileListResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	client, diags := providerdata.ConfigureProClassic(ctx, req.ProviderData, minJamfProVersion, "jamfplatform_pro_mobile_device_provisioning_profile")
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = client
}

// ListResourceConfigSchema describes the supported list filters.
func (r *ProvisioningProfileListResource) ListResourceConfigSchema(ctx context.Context, req list.ListResourceSchemaRequest, resp *list.ListResourceSchemaResponse) {
	resp.Schema = listschema.Schema{
		Description: "Lists Jamf Pro mobile device provisioning profiles. Supply an optional case-insensitive `name_substring` filter; filtering is applied client-side after the full list is fetched.",
		Attributes: map[string]listschema.Attribute{
			"filter": filters.ClassicListFilterAttribute(),
		},
	}
}

// List executes the query and streams profile identities back to Terraform.
func (r *ProvisioningProfileListResource) List(ctx context.Context, req list.ListRequest, stream *list.ListResultsStream) {
	if r.client == nil {
		stream.Results = list.ListResultsStreamDiagnostics(diag.Diagnostics{
			diag.NewErrorDiagnostic(
				"Unconfigured Provider",
				"The provider has not been configured yet. Re-run the command after `terraform init` completes successfully.",
			),
		})
		return
	}

	var config ProvisioningProfileListResourceModel
	diags := req.Config.Get(ctx, &config)
	if diags.HasError() {
		stream.Results = list.ListResultsStreamDiagnostics(diags)
		return
	}

	listCtx, cancel := context.WithTimeout(ctx, defaultListTimeout)
	defer cancel()

	resp, err := r.client.ListMobileDeviceProvisioningProfiles(listCtx)
	if err != nil {
		stream.Results = list.ListResultsStreamDiagnostics(diag.Diagnostics{
			diag.NewErrorDiagnostic("Unable to list Jamf Pro mobile device provisioning profiles", err.Error()),
		})
		return
	}

	items := []proclassic.MobileDeviceProvisioningProfilesItemMobileDeviceProvisioningProfile{}
	if resp != nil {
		items = resp.MobileDeviceProvisioningProfiles
	}

	filter := filters.ClassicFilterModel{}
	if config.Filter != nil {
		filter = *config.Filter
	}
	items = filters.ApplyClassicFilter(items, filter, provisioningProfileItemName)

	maxResults := req.Limit
	if maxResults <= 0 || maxResults > int64(len(items)) {
		maxResults = int64(len(items))
	}

	results := make([]list.ListResult, 0, maxResults)

	for _, p := range items {
		if int64(len(results)) >= maxResults {
			break
		}

		result := req.NewListResult(ctx)
		result.DisplayName = derefString(p.Name)

		id := helpers.StringValueFromIntPtr(p.ID)
		result.Diagnostics.Append(helpers.SetIdentity(ctx, result.Identity, provisioningProfileIdentityModel{ID: id})...)
		if result.Diagnostics.HasError() {
			stream.Results = list.ListResultsStreamDiagnostics(result.Diagnostics)
			return
		}

		if req.IncludeResource {
			state := ProvisioningProfileResourceModel{
				ID:                  id,
				Name:                helpers.StringPointerValueOrNull(p.Name),
				DisplayName:         helpers.StringPointerValueOrNull(p.DisplayName),
				ProfileData:         types.StringNull(),
				UUID:                helpers.StringPointerValueOrNull(p.UUID),
				CreationDateUTC:     types.StringNull(),
				CreationDateEpoch:   types.StringNull(),
				ExpirationDateUTC:   types.StringNull(),
				ExpirationDateEpoch: types.StringNull(),
				Timeouts:            helpers.NewResourceTimeoutsNullValue(provisioningProfileTimeoutAttributeTypes),
			}
			result.Diagnostics.Append(result.Resource.Set(ctx, &state)...)
			if result.Diagnostics.HasError() {
				stream.Results = list.ListResultsStreamDiagnostics(result.Diagnostics)
				return
			}
		}

		results = append(results, result)
	}

	tflog.Debug(ctx, "Listed Jamf Pro mobile device provisioning profiles", map[string]any{
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

// provisioningProfileItemName is the name accessor passed to filters.ApplyClassicFilter.
func provisioningProfileItemName(p proclassic.MobileDeviceProvisioningProfilesItemMobileDeviceProvisioningProfile) string {
	return derefString(p.Name)
}
