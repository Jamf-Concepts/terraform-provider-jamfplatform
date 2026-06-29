// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package account

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

const defaultListTimeout = 90 * time.Second

var _ list.ListResource = &AccountListResource{}
var _ list.ListResourceWithConfigure = &AccountListResource{}

// NewAccountListResource returns a list resource for account queries.
func NewAccountListResource() list.ListResource {
	return &AccountListResource{}
}

// AccountListResource implements query list support for admin accounts via the
// Pro v1 /accounts endpoint. The optional `filter` block is applied client-side
// on the username after the full list is fetched.
type AccountListResource struct {
	client *pro.Client
}

// Metadata sets the list resource type name.
func (r *AccountListResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pro_account"
}

// Configure wires the Jamf Pro client into the list resource.
func (r *AccountListResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	client, diags := providerdata.ConfigurePro(ctx, req.ProviderData, minJamfProVersion, "jamfplatform_pro_account")
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = client
}

// ListResourceConfigSchema describes the supported list filters.
func (r *AccountListResource) ListResourceConfigSchema(ctx context.Context, req list.ListResourceSchemaRequest, resp *list.ListResourceSchemaResponse) {
	resp.Schema = listschema.Schema{
		Description: "Lists Jamf Pro administrator accounts. Supply an optional case-insensitive `name_substring` filter on the username; filtering is applied client-side after the full list is fetched." + listResourcePrivileges,
		Attributes: map[string]listschema.Attribute{
			"filter": filters.ClassicListFilterAttribute(),
		},
	}
}

// List executes the query and streams account identities to Terraform.
func (r *AccountListResource) List(ctx context.Context, req list.ListRequest, stream *list.ListResultsStream) {
	if r.client == nil {
		stream.Results = list.ListResultsStreamDiagnostics(diag.Diagnostics{
			diag.NewErrorDiagnostic("Unconfigured Provider", "The provider has not been configured yet."),
		})
		return
	}

	var config AccountListResourceModel
	diags := req.Config.Get(ctx, &config)
	if diags.HasError() {
		stream.Results = list.ListResultsStreamDiagnostics(diags)
		return
	}

	listCtx, cancel := context.WithTimeout(ctx, defaultListTimeout)
	defer cancel()

	accounts, err := r.client.ListAccountsV1(listCtx, nil, "")
	if err != nil {
		stream.Results = list.ListResultsStreamDiagnostics(diag.Diagnostics{
			diag.NewErrorDiagnostic("Unable to list Jamf Pro accounts", err.Error()),
		})
		return
	}

	filter := filters.ClassicFilterModel{}
	if config.Filter != nil {
		filter = *config.Filter
	}
	items := filters.ApplyClassicFilter(accounts, filter, accountUsername)

	maxResults := req.Limit
	if maxResults <= 0 || maxResults > int64(len(items)) {
		maxResults = int64(len(items))
	}

	results := make([]list.ListResult, 0, maxResults)
	for i := range items {
		if int64(len(results)) >= maxResults {
			break
		}
		a := items[i]
		id := helpers.StringPointerValueOrNull(a.ID)
		result := req.NewListResult(ctx)
		result.DisplayName = accountUsername(a)
		result.Diagnostics.Append(helpers.SetIdentity(ctx, result.Identity, accountIdentityModel{ID: id})...)
		if result.Diagnostics.HasError() {
			stream.Results = list.ListResultsStreamDiagnostics(result.Diagnostics)
			return
		}

		if req.IncludeResource {
			state := AccountResourceModel{Timeouts: helpers.NewResourceTimeoutsNullValue(accountTimeoutAttributeTypes)}
			assignProBaseFields(&state, &a)
			state.PasswordWOVersion = types.Int64Null()
			result.Diagnostics.Append(result.Resource.Set(ctx, &state)...)
			if result.Diagnostics.HasError() {
				stream.Results = list.ListResultsStreamDiagnostics(result.Diagnostics)
				return
			}
		}

		results = append(results, result)
	}

	tflog.Debug(ctx, "Listed Jamf Pro accounts", map[string]any{
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

// accountUsername is the username accessor for the client-side filter.
func accountUsername(a pro.UserAccount) string {
	if a.Username == nil {
		return ""
	}
	return *a.Username
}
