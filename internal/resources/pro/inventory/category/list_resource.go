// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package category

import (
	"context"
	"fmt"

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
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/inventory/categories"
)

var _ list.ListResource = &CategoryListResource{}
var _ list.ListResourceWithConfigure = &CategoryListResource{}

// NewCategoryListResource returns a list resource for Jamf Pro category queries.
func NewCategoryListResource() list.ListResource {
	return &CategoryListResource{}
}

// CategoryListResource implements Terraform query list support for Jamf Pro categories.
type CategoryListResource struct {
	client *pro.Client
}

// Metadata sets the list resource type name.
func (r *CategoryListResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pro_category"
}

// Configure wires the Jamf Pro client into the list resource. Same shape as the resource
// and singular data source Configure: always fetch, swallow fetch errors only when the
// per-resource gate is empty, surface the floor warning when applicable.
func (r *CategoryListResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	pd, ok := req.ProviderData.(*providerdata.Data)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected List Configure Type",
			"Expected *providerdata.Data. Please report this issue to the provider developers.",
		)
		return
	}
	r.client = pro.New(pd.Client)

	version, err := pd.GetJamfProVersion(ctx)
	if err != nil {
		if minJamfProVersion == "" {
			return
		}
		resp.Diagnostics.AddError(
			"Failed to read Jamf Pro tenant version",
			fmt.Sprintf("jamfplatform_pro_category list requires Jamf Pro; could not read version: %s", err),
		)
		return
	}
	if minJamfProVersion != "" {
		resp.Diagnostics.Append(
			helpers.RequireMinJamfProVersion(version, minJamfProVersion, "jamfplatform_pro_category")...,
		)
	}
	if warn := pd.MaybeProviderFloorWarning(); warn != nil {
		resp.Diagnostics.Append(warn)
	}
}

// ListResourceConfigSchema describes the supported list filters.
func (r *CategoryListResource) ListResourceConfigSchema(ctx context.Context, req list.ListResourceSchemaRequest, resp *list.ListResourceSchemaResponse) {
	resp.Schema = listschema.Schema{
		Description: "Searches for Jamf Pro categories using the same filter clauses as the categories data source.",
		Attributes: map[string]listschema.Attribute{
			"filter": filters.ListFilterAttribute(
				filters.SelectorDescription(categories.CategoryFilterSelectors),
				categories.CategoryFilterSelectors,
			),
		},
	}
}

// List executes the query and streams category identities back to Terraform.
func (r *CategoryListResource) List(ctx context.Context, req list.ListRequest, stream *list.ListResultsStream) {
	if r.client == nil {
		stream.Results = list.ListResultsStreamDiagnostics(diag.Diagnostics{
			diag.NewErrorDiagnostic(
				"Unconfigured Provider",
				"The provider has not been configured yet. Re-run the command after `terraform init` completes successfully.",
			),
		})
		return
	}

	var config CategoryListResourceModel
	diags := req.Config.Get(ctx, &config)
	if diags.HasError() {
		stream.Results = list.ListResultsStreamDiagnostics(diags)
		return
	}

	filterExpression := filters.BuildRSQLExpression(config.Filters, filters.AllowList(categories.CategoryFilterSelectors))
	tflog.Debug(ctx, "category list filters", map[string]any{"filter": filterExpression})

	cats, err := r.client.ListCategoriesV1(ctx, nil, filterExpression)
	if err != nil {
		stream.Results = list.ListResultsStreamDiagnostics(diag.Diagnostics{
			diag.NewErrorDiagnostic("Unable to list Jamf Pro categories", err.Error()),
		})
		return
	}

	maxResults := req.Limit
	if maxResults <= 0 || maxResults > int64(len(cats)) {
		maxResults = int64(len(cats))
	}

	results := make([]list.ListResult, 0, int(maxResults))
	var emitted int64

	for _, c := range cats {
		if maxResults > 0 && emitted >= maxResults {
			break
		}

		result := req.NewListResult(ctx)
		result.DisplayName = c.Name

		id := helpers.StringPointerValueOrNull(c.ID)
		result.Diagnostics.Append(helpers.SetIdentity(ctx, result.Identity, categoryIdentityModel{ID: id})...)
		if result.Diagnostics.HasError() {
			stream.Results = list.ListResultsStreamDiagnostics(result.Diagnostics)
			return
		}

		if req.IncludeResource {
			state := CategoryResourceModel{
				ID:       id,
				Name:     types.StringValue(c.Name),
				Priority: types.Int64Value(int64(c.Priority)),
				Timeouts: helpers.NewResourceTimeoutsNullValue(categoryTimeoutAttributeTypes),
			}
			result.Diagnostics.Append(result.Resource.Set(ctx, &state)...)
			if result.Diagnostics.HasError() {
				stream.Results = list.ListResultsStreamDiagnostics(result.Diagnostics)
				return
			}
		}

		results = append(results, result)
		emitted++
	}

	tflog.Debug(ctx, "Listed Jamf Pro categories", map[string]any{
		"filter":   filterExpression,
		"limit":    req.Limit,
		"returned": len(results),
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
