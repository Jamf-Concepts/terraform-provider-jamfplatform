// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

// Package api_role_privileges implements the read-only
// jamfplatform_pro_api_role_privileges data source, which surfaces the set of
// valid Jamf Pro privilege strings for the tenant (the values accepted by
// jamfplatform_pro_api_role.privileges). The valid set varies by Jamf Pro
// version, so this is sourced live rather than from a static list.
package api_role_privileges

import (
	"context"
	"time"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/pro"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/datasource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/providerdata"
)

const defaultReadTimeout = 60 * time.Second

// minJamfProVersion is the minimum Jamf Pro tenant version required by this data
// source. Empty: no per-resource floor; the API Roles endpoints predate the
// provider's overall floor (11.0.0).
const minJamfProVersion = ""

// ApiRolePrivilegesDataSourceModel is the data source model for the privilege list.
type ApiRolePrivilegesDataSourceModel struct {
	ID         types.String   `tfsdk:"id"`
	Search     types.String   `tfsdk:"search"`
	Privileges types.Set      `tfsdk:"privileges"`
	Timeouts   timeouts.Value `tfsdk:"timeouts"`
}

// ApiRolePrivilegesDataSource implements the Terraform data source for Jamf Pro API role privileges.
type ApiRolePrivilegesDataSource struct {
	client *pro.Client
}

var _ datasource.DataSource = &ApiRolePrivilegesDataSource{}

// NewApiRolePrivilegesDataSource returns a new instance of ApiRolePrivilegesDataSource.
func NewApiRolePrivilegesDataSource() datasource.DataSource {
	return &ApiRolePrivilegesDataSource{}
}

// Metadata sets the data source type name for the Terraform provider.
func (d *ApiRolePrivilegesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pro_api_role_privileges"
}

// Schema returns the data source schema.
func (d *ApiRolePrivilegesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads the set of valid Jamf Pro API role privilege strings for the tenant — the values accepted by `jamfplatform_pro_api_role.privileges`. The valid set varies by Jamf Pro version. Use `search` to narrow the result to privileges whose name contains a substring." + dataSourcePrivileges,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Internal identifier for this data source read.",
				Computed:            true,
			},
			"search": schema.StringAttribute{
				MarkdownDescription: "Optional case-insensitive substring to narrow the returned privileges. When omitted, the full privilege list is returned.",
				Optional:            true,
			},
			"privileges": schema.SetAttribute{
				MarkdownDescription: "The set of valid Jamf Pro privilege strings.",
				ElementType:         types.StringType,
				Computed:            true,
			},
			"timeouts": timeouts.Attributes(ctx),
		},
	}
}

// Configure wires the Jamf Pro client into the data source via the shared
// providerdata.ConfigurePro helper.
func (d *ApiRolePrivilegesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	client, diags := providerdata.ConfigurePro(ctx, req.ProviderData, minJamfProVersion, "jamfplatform_pro_api_role_privileges")
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	d.client = client
}

// Read fetches the privilege list (optionally filtered by search) and populates state.
func (d *ApiRolePrivilegesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.client == nil {
		resp.Diagnostics.AddError(
			"Provider not configured",
			"The provider client was not configured. Please ensure the provider block is set up correctly.",
		)
		return
	}

	var data ApiRolePrivilegesDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	readTimeout, timeoutDiags := helpers.ResolveTimeout(ctx, data.Timeouts.IsNull(), data.Timeouts.IsUnknown(), defaultReadTimeout, data.Timeouts.Read)
	resp.Diagnostics.Append(timeoutDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	readCtx, cancel := context.WithTimeout(ctx, readTimeout)
	defer cancel()

	var privileges *pro.ApiRolePrivileges
	var err error
	if search := data.Search.ValueString(); !data.Search.IsNull() && search != "" {
		privileges, err = d.client.SearchApiRolePrivilegesV1(readCtx, search, "")
	} else {
		privileges, err = d.client.ListApiRolePrivilegesV1(readCtx)
	}
	if err != nil {
		resp.Diagnostics.AddError("Unable to list Jamf Pro API role privileges", err.Error())
		return
	}

	set, setDiags := types.SetValueFrom(ctx, types.StringType, privileges.Privileges)
	resp.Diagnostics.Append(setDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Privileges = set
	data.ID = types.StringValue("api_role_privileges")

	tflog.Trace(ctx, "read Jamf Pro API role privileges data source", map[string]any{"count": len(privileges.Privileges)})
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
