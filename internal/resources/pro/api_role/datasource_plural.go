// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package api_role

import (
	"context"
	"time"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/pro"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/datasource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/filters"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/providerdata"
)

const defaultPluralReadTimeout = 90 * time.Second

// ApiRoleFilterSelectors enumerates the RSQL selectors accepted by the api-roles endpoint.
var ApiRoleFilterSelectors = []string{
	"id",
	"displayName",
}

// ApiRolesDataSource implements the Terraform data source for Jamf Pro API role searches.
type ApiRolesDataSource struct {
	client *pro.Client
}

var _ datasource.DataSource = &ApiRolesDataSource{}

// NewApiRolesDataSource returns a new instance of ApiRolesDataSource.
func NewApiRolesDataSource() datasource.DataSource {
	return &ApiRolesDataSource{}
}

// Metadata sets the data source type name for the Terraform provider.
func (d *ApiRolesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pro_api_roles"
}

// Schema returns the plural data source schema.
func (d *ApiRolesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Search Jamf Pro API roles using optional RSQL filters on `id` and `displayName`." + pluralDataSourcePrivileges,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Internal identifier for this data source read.",
				Computed:            true,
			},
			"timeouts": timeouts.Attributes(ctx),
			"filter": filters.FilterAttribute(
				filters.SelectorDescription(ApiRoleFilterSelectors),
				ApiRoleFilterSelectors,
			),
			"api_roles": schema.ListNestedAttribute{
				MarkdownDescription: "API roles matching the supplied filter.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "API role ID assigned by Jamf Pro.",
							Computed:            true,
						},
						"display_name": schema.StringAttribute{
							MarkdownDescription: "Role display name.",
							Computed:            true,
						},
						"privileges": schema.SetAttribute{
							MarkdownDescription: "The set of Jamf Pro privilege strings granted by this role.",
							ElementType:         types.StringType,
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

// Configure wires the Jamf Pro client into the data source via the shared
// providerdata.ConfigurePro helper.
func (d *ApiRolesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	client, diags := providerdata.ConfigurePro(ctx, req.ProviderData, minJamfProVersion, "jamfplatform_pro_api_roles")
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	d.client = client
}

// Read fetches API roles matching the supplied filters and populates state.
func (d *ApiRolesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.client == nil {
		resp.Diagnostics.AddError(
			"Provider not configured",
			"The provider client was not configured. Please ensure the provider block is set up correctly.",
		)
		return
	}

	var data ApiRolesDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	readTimeout, timeoutDiags := helpers.ResolveTimeout(ctx, data.Timeouts.IsNull(), data.Timeouts.IsUnknown(), defaultPluralReadTimeout, data.Timeouts.Read)
	resp.Diagnostics.Append(timeoutDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	readCtx, cancel := context.WithTimeout(ctx, readTimeout)
	defer cancel()

	filterExpression := filters.BuildRSQLExpression(data.Filters, filters.AllowList(ApiRoleFilterSelectors))
	tflog.Debug(ctx, "api roles filter expression", map[string]any{"filter": filterExpression})

	roles, err := d.client.ListApiRolesV1(readCtx, nil, filterExpression)
	if err != nil {
		resp.Diagnostics.AddError("Unable to list Jamf Pro API roles", err.Error())
		return
	}

	results := make([]ApiRolesDataSourceResultModel, 0, len(roles))
	for _, role := range roles {
		privileges, privDiags := types.SetValueFrom(ctx, types.StringType, role.Privileges)
		resp.Diagnostics.Append(privDiags...)
		if resp.Diagnostics.HasError() {
			return
		}
		results = append(results, ApiRolesDataSourceResultModel{
			ID:          types.StringValue(role.ID),
			DisplayName: types.StringValue(role.DisplayName),
			Privileges:  privileges,
		})
	}

	data.ApiRoles = results
	data.ID = types.StringValue("api_roles")

	tflog.Trace(ctx, "listed Jamf Pro API roles data source", map[string]any{
		"filter": filterExpression,
		"count":  len(results),
	})
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
