// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package class

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

// defaultListTimeout caps how long the list operation waits on the classic
// /classes endpoint.
const defaultListTimeout = 90 * time.Second

var _ list.ListResource = &ClassListResource{}
var _ list.ListResourceWithConfigure = &ClassListResource{}

// NewClassListResource returns a list resource for class queries.
func NewClassListResource() list.ListResource {
	return &ClassListResource{}
}

// ClassListResource implements Terraform query list support. Classic /classes
// has no RSQL — the optional `filter` block is applied client-side via
// filters.ApplyClassicFilter after the full list is fetched. List items carry
// only id, name, and description on the wire; the membership attributes are set
// to null on list results (read the full resource for membership).
type ClassListResource struct {
	client *proclassic.Client
}

// ClassListResourceModel is the config model for list queries.
type ClassListResourceModel struct {
	Filter *filters.ClassicFilterModel `tfsdk:"filter"`
}

// Metadata sets the list resource type name.
func (r *ClassListResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pro_class"
}

// Configure wires the Jamf ProClassic client into the list resource.
func (r *ClassListResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	client, diags := providerdata.ConfigureProClassic(ctx, req.ProviderData, minJamfProVersion, "jamfplatform_pro_class")
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = client
}

// ListResourceConfigSchema describes the supported list filters.
func (r *ClassListResource) ListResourceConfigSchema(ctx context.Context, req list.ListResourceSchemaRequest, resp *list.ListResourceSchemaResponse) {
	resp.Schema = listschema.Schema{
		Description: "Lists Jamf Pro classes. Supply an optional case-insensitive `name_substring` filter applied client-side after the full list is fetched.",
		Attributes: map[string]listschema.Attribute{
			"filter": filters.ClassicListFilterAttribute(),
		},
	}
}

// List executes the query and streams class identities back to Terraform.
func (r *ClassListResource) List(ctx context.Context, req list.ListRequest, stream *list.ListResultsStream) {
	if r.client == nil {
		stream.Results = list.ListResultsStreamDiagnostics(diag.Diagnostics{
			diag.NewErrorDiagnostic(
				"Unconfigured Provider",
				"The provider has not been configured yet. Re-run the command after `terraform init` completes successfully.",
			),
		})
		return
	}

	var config ClassListResourceModel
	diags := req.Config.Get(ctx, &config)
	if diags.HasError() {
		stream.Results = list.ListResultsStreamDiagnostics(diags)
		return
	}

	listCtx, cancel := context.WithTimeout(ctx, defaultListTimeout)
	defer cancel()

	resp, err := r.client.ListClasses(listCtx)
	if err != nil {
		stream.Results = list.ListResultsStreamDiagnostics(diag.Diagnostics{
			diag.NewErrorDiagnostic("Unable to list Jamf Pro classes", err.Error()),
		})
		return
	}

	items := []proclassic.ClassesItemClass{}
	if resp != nil {
		items = resp.Classes
	}

	filter := filters.ClassicFilterModel{}
	if config.Filter != nil {
		filter = *config.Filter
	}
	items = filters.ApplyClassicFilter(items, filter, classListItemName)

	maxResults := req.Limit
	if maxResults <= 0 || maxResults > int64(len(items)) {
		maxResults = int64(len(items))
	}

	results := make([]list.ListResult, 0, maxResults)

	for _, c := range items {
		if int64(len(results)) >= maxResults {
			break
		}

		result := req.NewListResult(ctx)
		result.DisplayName = derefString(c.Name)

		id := helpers.StringValueFromIntPtr(c.ID)
		result.Diagnostics.Append(helpers.SetIdentity(ctx, result.Identity, classIdentityModel{ID: id})...)
		if result.Diagnostics.HasError() {
			stream.Results = list.ListResultsStreamDiagnostics(result.Diagnostics)
			return
		}

		if req.IncludeResource {
			// List response carries id, name, and description only. Every membership
			// attribute is null on list results.
			state := ClassResourceModel{
				ID:                   id,
				Name:                 helpers.StringPointerValueOrNull(c.Name),
				Description:          helpers.StringPointerValueOrNull(c.Description),
				SiteID:               types.StringNull(),
				SiteName:             types.StringNull(),
				Source:               types.StringNull(),
				Students:             types.SetNull(types.StringType),
				Teachers:             types.SetNull(types.StringType),
				StudentGroupIDs:      types.SetNull(types.StringType),
				TeacherGroupIDs:      types.SetNull(types.StringType),
				MobileDeviceGroupIDs: types.SetNull(types.StringType),
				StudentIDs:           types.SetNull(types.StringType),
				TeacherIDs:           types.SetNull(types.StringType),
				Timeouts:             helpers.NewResourceTimeoutsNullValue(classTimeoutAttributeTypes),
			}
			result.Diagnostics.Append(result.Resource.Set(ctx, &state)...)
			if result.Diagnostics.HasError() {
				stream.Results = list.ListResultsStreamDiagnostics(result.Diagnostics)
				return
			}
		}

		results = append(results, result)
	}

	tflog.Debug(ctx, "Listed Jamf Pro classes", map[string]any{
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

// classListItemName is the name accessor passed to filters.ApplyClassicFilter.
func classListItemName(c proclassic.ClassesItemClass) string {
	return derefString(c.Name)
}

// derefString returns the underlying string for a non-nil *string, or "" for nil.
func derefString(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
