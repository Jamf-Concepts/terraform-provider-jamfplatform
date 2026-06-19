// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package account_group

import (
	"context"
	"time"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/proclassic"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/list"
	listschema "github.com/hashicorp/terraform-plugin-framework/list/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/filters"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/providerdata"
)

// defaultListTimeout caps the classic /accounts list + per-item fetches.
const defaultListTimeout = 120 * time.Second

var _ list.ListResource = &AccountGroupListResource{}
var _ list.ListResourceWithConfigure = &AccountGroupListResource{}

// NewAccountGroupListResource returns a list resource for account group queries.
func NewAccountGroupListResource() list.ListResource {
	return &AccountGroupListResource{}
}

// AccountGroupListResource implements query list support for account groups.
// The classic /accounts list returns id+name only; the optional `filter` block
// is applied client-side on the group name. When IncludeResource is requested,
// each matching group is fetched individually (classic) and hydrated with
// import semantics.
type AccountGroupListResource struct {
	client *proclassic.Client
}

// Metadata sets the list resource type name.
func (r *AccountGroupListResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pro_account_group"
}

// Configure wires the Jamf ProClassic client into the list resource.
func (r *AccountGroupListResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	client, diags := providerdata.ConfigureProClassic(ctx, req.ProviderData, minJamfProVersion, "jamfplatform_pro_account_group")
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = client
}

// ListResourceConfigSchema describes the supported list filters.
func (r *AccountGroupListResource) ListResourceConfigSchema(ctx context.Context, req list.ListResourceSchemaRequest, resp *list.ListResourceSchemaResponse) {
	resp.Schema = listschema.Schema{
		Description: "Lists Jamf Pro administrator account groups. Supply an optional case-insensitive `name_substring` filter; filtering is applied client-side after the full list is fetched.",
		Attributes: map[string]listschema.Attribute{
			"filter": filters.ClassicListFilterAttribute(),
		},
	}
}

// List executes the query and streams account group identities to Terraform.
func (r *AccountGroupListResource) List(ctx context.Context, req list.ListRequest, stream *list.ListResultsStream) {
	if r.client == nil {
		stream.Results = list.ListResultsStreamDiagnostics(diag.Diagnostics{
			diag.NewErrorDiagnostic("Unconfigured Provider", "The provider has not been configured yet."),
		})
		return
	}

	var config AccountGroupListResourceModel
	diags := req.Config.Get(ctx, &config)
	if diags.HasError() {
		stream.Results = list.ListResultsStreamDiagnostics(diags)
		return
	}

	listCtx, cancel := context.WithTimeout(ctx, defaultListTimeout)
	defer cancel()

	resp, err := r.client.ListAccounts(listCtx)
	if err != nil {
		stream.Results = list.ListResultsStreamDiagnostics(diag.Diagnostics{
			diag.NewErrorDiagnostic("Unable to list Jamf Pro account groups", err.Error()),
		})
		return
	}

	items := []proclassic.AccountsGroupsGroupItem{}
	if resp != nil && resp.Groups != nil && resp.Groups.Group != nil {
		items = *resp.Groups.Group
	}

	filter := filters.ClassicFilterModel{}
	if config.Filter != nil {
		filter = *config.Filter
	}
	items = filters.ApplyClassicFilter(items, filter, groupItemName)

	maxResults := req.Limit
	if maxResults <= 0 || maxResults > int64(len(items)) {
		maxResults = int64(len(items))
	}

	results := make([]list.ListResult, 0, maxResults)
	for _, g := range items {
		if int64(len(results)) >= maxResults {
			break
		}

		id := helpers.StringValueFromIntPtr(g.ID)
		result := req.NewListResult(ctx)
		result.DisplayName = helpers.DerefString(g.Name)
		result.Diagnostics.Append(helpers.SetIdentity(ctx, result.Identity, accountGroupIdentityModel{ID: id})...)
		if result.Diagnostics.HasError() {
			stream.Results = list.ListResultsStreamDiagnostics(result.Diagnostics)
			return
		}

		if req.IncludeResource {
			state := AccountGroupResourceModel{
				ID:       id,
				Timeouts: helpers.NewResourceTimeoutsNullValue(accountGroupTimeoutAttributeTypes),
			}
			got, err := r.client.GetAccountGroupByID(listCtx, id.ValueString())
			if err != nil {
				result.Diagnostics.AddError("Unable to read Jamf Pro account group", err.Error())
				stream.Results = list.ListResultsStreamDiagnostics(result.Diagnostics)
				return
			}
			result.Diagnostics.Append(assignAccountGroupResourceModel(listCtx, &state, got, true)...)
			result.Diagnostics.Append(result.Resource.Set(ctx, &state)...)
			if result.Diagnostics.HasError() {
				stream.Results = list.ListResultsStreamDiagnostics(result.Diagnostics)
				return
			}
		}

		results = append(results, result)
	}

	tflog.Debug(ctx, "Listed Jamf Pro account groups", map[string]any{
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

// groupItemName is the name accessor for the classic group list filter.
func groupItemName(g proclassic.AccountsGroupsGroupItem) string { return helpers.DerefString(g.Name) }
