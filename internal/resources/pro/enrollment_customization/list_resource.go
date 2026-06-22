// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

// SDK endpoints used:
//   pro.ListEnrollmentCustomizationsV2
// Status: current. Last reviewed 2026-05-28.

package enrollment_customization

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

// defaultListTimeout caps how long the list operation will wait. The Pro v2
// list endpoint is paginated client-side by the SDK; 90 seconds is more than
// enough for tenants with hundreds of customizations.
const defaultListTimeout = 90 * time.Second

var (
	_ list.ListResource              = &EnrollmentCustomizationListResource{}
	_ list.ListResourceWithConfigure = &EnrollmentCustomizationListResource{}
)

// EnrollmentCustomizationListResource implements Terraform query list support
// for Jamf Pro enrollment customizations. The Pro v2 list endpoint does not
// accept an RSQL filter, so the optional `filter` block is applied
// client-side after the full list is fetched.
type EnrollmentCustomizationListResource struct {
	client *pro.Client
}

// NewEnrollmentCustomizationListResource returns a list resource for Jamf Pro
// enrollment customization queries.
func NewEnrollmentCustomizationListResource() list.ListResource {
	return &EnrollmentCustomizationListResource{}
}

// Metadata sets the list resource type name.
func (r *EnrollmentCustomizationListResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pro_enrollment_customization"
}

// Configure wires the Jamf Pro client into the list resource via the shared
// providerdata.ConfigurePro helper.
func (r *EnrollmentCustomizationListResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	client, diags := providerdata.ConfigurePro(ctx, req.ProviderData, minJamfProVersion, "jamfplatform_pro_enrollment_customization")
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = client
}

// ListResourceConfigSchema describes the supported list filters.
func (r *EnrollmentCustomizationListResource) ListResourceConfigSchema(ctx context.Context, req list.ListResourceSchemaRequest, resp *list.ListResourceSchemaResponse) {
	resp.Schema = listschema.Schema{
		Description: "Lists Jamf Pro enrollment customizations. Supply an optional case-insensitive `name_substring` filter applied client-side after the full list is fetched. The list response carries the parent record (display name, description, site, branding palette + icon URL); panes are not included — read them via the singular resource per ID.",
		Attributes: map[string]listschema.Attribute{
			"filter": filters.ClassicListFilterAttribute(),
		},
	}
}

// List executes the query and streams customization identities back to
// Terraform.
func (r *EnrollmentCustomizationListResource) List(ctx context.Context, req list.ListRequest, stream *list.ListResultsStream) {
	if r.client == nil {
		stream.Results = list.ListResultsStreamDiagnostics(diag.Diagnostics{
			diag.NewErrorDiagnostic(
				"Unconfigured Provider",
				"The provider has not been configured yet. Re-run the command after `terraform init` completes successfully.",
			),
		})
		return
	}

	var config EnrollmentCustomizationListResourceModel
	diags := req.Config.Get(ctx, &config)
	if diags.HasError() {
		stream.Results = list.ListResultsStreamDiagnostics(diags)
		return
	}

	listCtx, cancel := context.WithTimeout(ctx, defaultListTimeout)
	defer cancel()

	items, err := r.client.ListEnrollmentCustomizationsV2(listCtx, nil)
	if err != nil {
		stream.Results = list.ListResultsStreamDiagnostics(diag.Diagnostics{
			diag.NewErrorDiagnostic("Unable to list Jamf Pro enrollment customizations", err.Error()),
		})
		return
	}

	filter := filters.ClassicFilterModel{}
	if config.Filter != nil {
		filter = *config.Filter
	}
	items = filters.ApplyClassicFilter(items, filter, enrollmentCustomizationListItemName)

	maxResults := req.Limit
	if maxResults <= 0 || maxResults > int64(len(items)) {
		maxResults = int64(len(items))
	}

	results := make([]list.ListResult, 0, maxResults)

	for i := range items {
		if int64(len(results)) >= maxResults {
			break
		}
		item := items[i]

		result := req.NewListResult(ctx)
		result.DisplayName = item.DisplayName

		var id types.String
		if item.ID != nil {
			id = types.StringValue(*item.ID)
		} else {
			id = types.StringNull()
		}
		result.Diagnostics.Append(helpers.SetIdentity(ctx, result.Identity, EnrollmentCustomizationIdentityModel{ID: id})...)
		if result.Diagnostics.HasError() {
			stream.Results = list.ListResultsStreamDiagnostics(result.Diagnostics)
			return
		}

		if req.IncludeResource {
			// IncludeResource targets the resource schema. The list endpoint
			// only carries the parent record; panes stay null in the result
			// state — admins who need pane-level detail should follow up with
			// a singular resource read by ID.
			state := EnrollmentCustomizationResourceModel{
				Timeouts: helpers.NewResourceTimeoutsNullValue(enrollmentCustomizationTimeoutAttributeTypes),
			}
			assignParentToResource(&state, &item)
			result.Diagnostics.Append(result.Resource.Set(ctx, &state)...)
			if result.Diagnostics.HasError() {
				stream.Results = list.ListResultsStreamDiagnostics(result.Diagnostics)
				return
			}
		}

		results = append(results, result)
	}

	tflog.Debug(ctx, "Listed Jamf Pro enrollment customizations", map[string]any{
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

// enrollmentCustomizationListItemName is the name accessor passed to
// filters.ApplyClassicFilter. The SDK list endpoint returns
// EnrollmentCustomizationV2 elements with a non-pointer DisplayName field.
func enrollmentCustomizationListItemName(item pro.EnrollmentCustomizationV2) string {
	return item.DisplayName
}
