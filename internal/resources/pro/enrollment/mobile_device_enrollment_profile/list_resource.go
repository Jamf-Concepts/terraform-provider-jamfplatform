// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package mobile_device_enrollment_profile

import (
	"context"
	"time"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/proclassic"
	"github.com/hashicorp/terraform-plugin-framework/attr"
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

const defaultListTimeout = 90 * time.Second

var _ list.ListResource = &EnrollmentProfileListResource{}
var _ list.ListResourceWithConfigure = &EnrollmentProfileListResource{}

func NewEnrollmentProfileListResource() list.ListResource {
	return &EnrollmentProfileListResource{}
}

type EnrollmentProfileListResource struct {
	client *proclassic.Client
}

func (r *EnrollmentProfileListResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pro_mobile_device_enrollment_profile"
}

func (r *EnrollmentProfileListResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	client, diags := providerdata.ConfigureProClassic(ctx, req.ProviderData, minJamfProVersion, "jamfplatform_pro_mobile_device_enrollment_profile")
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = client
}

func (r *EnrollmentProfileListResource) ListResourceConfigSchema(ctx context.Context, req list.ListResourceSchemaRequest, resp *list.ListResourceSchemaResponse) {
	resp.Schema = listschema.Schema{
		Description: "Lists Jamf Pro mobile device enrollment profiles. Supply an optional case-insensitive `name_substring` filter; filtering is applied client-side after the full list is fetched.",
		Attributes: map[string]listschema.Attribute{
			"filter": filters.ClassicListFilterAttribute(),
		},
	}
}

func (r *EnrollmentProfileListResource) List(ctx context.Context, req list.ListRequest, stream *list.ListResultsStream) {
	if r.client == nil {
		stream.Results = list.ListResultsStreamDiagnostics(diag.Diagnostics{
			diag.NewErrorDiagnostic("Unconfigured Provider", "The provider has not been configured yet."),
		})
		return
	}

	var config EnrollmentProfileListResourceModel
	diags := req.Config.Get(ctx, &config)
	if diags.HasError() {
		stream.Results = list.ListResultsStreamDiagnostics(diags)
		return
	}

	listCtx, cancel := context.WithTimeout(ctx, defaultListTimeout)
	defer cancel()

	apiResp, err := r.client.ListMobileDeviceEnrollmentProfiles(listCtx)
	if err != nil {
		stream.Results = list.ListResultsStreamDiagnostics(diag.Diagnostics{
			diag.NewErrorDiagnostic("Unable to list Jamf Pro mobile device enrollment profiles", err.Error()),
		})
		return
	}

	items := []proclassic.MobileDeviceEnrollmentProfilesItemMobileDeviceEnrollmentProfile{}
	if apiResp != nil {
		items = apiResp.MobileDeviceEnrollmentProfiles
	}

	filter := filters.ClassicFilterModel{}
	if config.Filter != nil {
		filter = *config.Filter
	}
	items = filters.ApplyClassicFilter(items, filter, enrollmentProfileItemName)

	maxResults := req.Limit
	if maxResults <= 0 || maxResults > int64(len(items)) {
		maxResults = int64(len(items))
	}

	results := make([]list.ListResult, 0, maxResults)
	emptyAttachments := types.ListValueMust(attachmentObjectType, []attr.Value{})

	for _, p := range items {
		if int64(len(results)) >= maxResults {
			break
		}
		result := req.NewListResult(ctx)
		result.DisplayName = helpers.DerefString(p.Name)

		id := helpers.StringValueFromIntPtr(p.ID)
		result.Diagnostics.Append(helpers.SetIdentity(ctx, result.Identity, enrollmentProfileIdentityModel{ID: id})...)
		if result.Diagnostics.HasError() {
			stream.Results = list.ListResultsStreamDiagnostics(result.Diagnostics)
			return
		}

		if req.IncludeResource {
			state := EnrollmentProfileResourceModel{
				ID:          id,
				Name:        helpers.StringPointerValueOrNull(p.Name),
				Description: types.StringNull(),
				SiteID:      types.StringNull(),
				SiteName:    types.StringNull(),
				Invitation:  bigIntStringOrNull(p.Invitation),
				UUID:        types.StringNull(),
				Attachments: emptyAttachments,
				Timeouts:    helpers.NewResourceTimeoutsNullValue(enrollmentProfileTimeoutAttributeTypes),
			}
			result.Diagnostics.Append(result.Resource.Set(ctx, &state)...)
			if result.Diagnostics.HasError() {
				stream.Results = list.ListResultsStreamDiagnostics(result.Diagnostics)
				return
			}
		}
		results = append(results, result)
	}

	tflog.Debug(ctx, "Listed Jamf Pro mobile device enrollment profiles", map[string]any{
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

func enrollmentProfileItemName(p proclassic.MobileDeviceEnrollmentProfilesItemMobileDeviceEnrollmentProfile) string {
	return helpers.DerefString(p.Name)
}
