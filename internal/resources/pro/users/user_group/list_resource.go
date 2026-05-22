// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package user_group

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
// /usergroups endpoint.
const defaultListTimeout = 90 * time.Second

var _ list.ListResource = &UserGroupListResource{}
var _ list.ListResourceWithConfigure = &UserGroupListResource{}

// NewUserGroupListResource returns a list resource for Jamf Pro user group queries.
func NewUserGroupListResource() list.ListResource {
	return &UserGroupListResource{}
}

// UserGroupListResource implements Terraform query list support for Jamf Pro
// user groups. Classic /usergroups has no RSQL — the optional `filter` block
// is applied client-side via filters.ApplyClassicFilter after the full list
// is fetched. List items carry only id, name, is_smart, is_notify_on_change
// on the wire (surfaced as group_type and notify_on_membership_change);
// every other resource attribute is set to null on list results.
type UserGroupListResource struct {
	client *proclassic.Client
}

// UserGroupListResourceModel is the config model for user group list queries.
type UserGroupListResourceModel struct {
	Filter *filters.ClassicFilterModel `tfsdk:"filter"`
}

// Metadata sets the list resource type name.
func (r *UserGroupListResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pro_user_group"
}

// Configure wires the Jamf ProClassic client into the list resource.
func (r *UserGroupListResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	client, diags := providerdata.ConfigureProClassic(ctx, req.ProviderData, minJamfProVersion, "jamfplatform_pro_user_group")
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = client
}

// ListResourceConfigSchema describes the supported list filters.
func (r *UserGroupListResource) ListResourceConfigSchema(ctx context.Context, req list.ListResourceSchemaRequest, resp *list.ListResourceSchemaResponse) {
	resp.Schema = listschema.Schema{
		Description: "Lists Jamf Pro user groups. Classic has no RSQL — supply an optional case-insensitive `name_substring` filter applied client-side after the full list is fetched.",
		Attributes: map[string]listschema.Attribute{
			"filter": filters.ClassicListFilterAttribute(),
		},
	}
}

// List executes the query and streams user group identities back to Terraform.
func (r *UserGroupListResource) List(ctx context.Context, req list.ListRequest, stream *list.ListResultsStream) {
	if r.client == nil {
		stream.Results = list.ListResultsStreamDiagnostics(diag.Diagnostics{
			diag.NewErrorDiagnostic(
				"Unconfigured Provider",
				"The provider has not been configured yet. Re-run the command after `terraform init` completes successfully.",
			),
		})
		return
	}

	var config UserGroupListResourceModel
	diags := req.Config.Get(ctx, &config)
	if diags.HasError() {
		stream.Results = list.ListResultsStreamDiagnostics(diags)
		return
	}

	listCtx, cancel := context.WithTimeout(ctx, defaultListTimeout)
	defer cancel()

	resp, err := r.client.ListUserGroups(listCtx)
	if err != nil {
		stream.Results = list.ListResultsStreamDiagnostics(diag.Diagnostics{
			diag.NewErrorDiagnostic("Unable to list Jamf Pro user groups", err.Error()),
		})
		return
	}

	items := []proclassic.UserGroupsItemUserGroup{}
	if resp != nil {
		items = resp.UserGroups
	}

	filter := filters.ClassicFilterModel{}
	if config.Filter != nil {
		filter = *config.Filter
	}
	items = filters.ApplyClassicFilter(items, filter, userGroupListItemName)

	maxResults := req.Limit
	if maxResults <= 0 || maxResults > int64(len(items)) {
		maxResults = int64(len(items))
	}

	results := make([]list.ListResult, 0, maxResults)

	for _, u := range items {
		if int64(len(results)) >= maxResults {
			break
		}

		result := req.NewListResult(ctx)
		result.DisplayName = derefString(u.Name)

		id := helpers.StringValueFromIntPtr(u.ID)
		result.Diagnostics.Append(helpers.SetIdentity(ctx, result.Identity, userGroupIdentityModel{ID: id})...)
		if result.Diagnostics.HasError() {
			stream.Results = list.ListResultsStreamDiagnostics(result.Diagnostics)
			return
		}

		if req.IncludeResource {
			// List response carries id, name, is_smart, is_notify_on_change.
			// Every other Optional/Computed attribute is null on list results.
			state := UserGroupResourceModel{
				ID:                       id,
				Name:                     helpers.StringPointerValueOrNull(u.Name),
				GroupType:                groupTypeFromIsSmart(u.IsSmart),
				NotifyOnMembershipChange: helpers.BoolPointerValueOrNull(u.IsNotifyOnChange),
				SiteID:                   types.StringNull(),
				SiteName:                 types.StringNull(),
				Criteria:                 nil,
				Members:                  types.SetNull(types.StringType),
				MemberCount:              types.Int64Null(),
				Timeouts:                 helpers.NewResourceTimeoutsNullValue(userGroupTimeoutAttributeTypes),
			}
			result.Diagnostics.Append(result.Resource.Set(ctx, &state)...)
			if result.Diagnostics.HasError() {
				stream.Results = list.ListResultsStreamDiagnostics(result.Diagnostics)
				return
			}
		}

		results = append(results, result)
	}

	tflog.Debug(ctx, "Listed Jamf Pro user groups", map[string]any{
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

// userGroupListItemName is the name accessor passed to filters.ApplyClassicFilter.
func userGroupListItemName(u proclassic.UserGroupsItemUserGroup) string {
	return derefString(u.Name)
}

// derefString returns the underlying string for a non-nil *string, or "" for nil.
func derefString(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
