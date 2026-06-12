// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package account_group

import (
	"context"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/pro"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/datasource/timeouts"
	datasourcevalidator "github.com/hashicorp/terraform-plugin-framework-validators/datasourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/providerdata"
)

// memberObjectAttrTypes is the shape of a data-source member entry (Pro v1).
var memberObjectAttrTypes = map[string]attr.Type{
	"id":       types.StringType,
	"username": types.StringType,
	"realname": types.StringType,
	"email":    types.StringType,
}

// AccountGroupDataSource implements the Pro v1 account-group data source.
type AccountGroupDataSource struct {
	client *pro.Client
}

var _ datasource.DataSource = &AccountGroupDataSource{}
var _ datasource.DataSourceWithConfigValidators = &AccountGroupDataSource{}

// NewAccountGroupDataSource returns a new instance of AccountGroupDataSource.
func NewAccountGroupDataSource() datasource.DataSource {
	return &AccountGroupDataSource{}
}

// Metadata sets the data source type name.
func (d *AccountGroupDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pro_account_group"
}

// ConfigValidators enforces exactly one of id / display_name.
func (d *AccountGroupDataSource) ConfigValidators(ctx context.Context) []datasource.ConfigValidator {
	return []datasource.ConfigValidator{
		datasourcevalidator.ExactlyOneOf(
			path.MatchRoot("id"),
			path.MatchRoot("display_name"),
		),
	}
}

// Schema returns the data source schema. Values reflect the Pro v1 JSON shape
// (flat privileges, Pro enum spellings such as `FullAccess`/`ADMINISTRATOR`),
// which differ from the resource's classic spellings.
func (d *AccountGroupDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Look up a Jamf Pro administrator account group by `id` or `display_name` via the Pro v1 `/account-groups` API. Values use the Pro JSON spellings (e.g. `FullAccess`, `ADMINISTRATOR`) and a flat privilege list, unlike the `jamfplatform_pro_account_group` resource (classic spellings, categorised privileges).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Account group ID. Provide this or `display_name`.",
				Optional:            true,
				Computed:            true,
			},
			"display_name": schema.StringAttribute{
				MarkdownDescription: "Account group display name. Provide this or `id`.",
				Optional:            true,
				Computed:            true,
			},
			"access_level": schema.StringAttribute{
				MarkdownDescription: "Access level (Pro spelling, e.g. `FullAccess`).",
				Computed:            true,
			},
			"privilege_level": schema.StringAttribute{
				MarkdownDescription: "Privilege set (Pro spelling, e.g. `ADMINISTRATOR`).",
				Computed:            true,
			},
			"site_id": schema.StringAttribute{
				MarkdownDescription: "Scoped site ID (`-1` for none).",
				Computed:            true,
			},
			"ldap_server_id": schema.StringAttribute{
				MarkdownDescription: "Backing LDAP / cloud-identity-provider server ID.",
				Computed:            true,
			},
			"members": schema.ListNestedAttribute{
				MarkdownDescription: "Account members of the group.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":       schema.StringAttribute{Computed: true, MarkdownDescription: "Account ID."},
						"username": schema.StringAttribute{Computed: true, MarkdownDescription: "Account username."},
						"realname": schema.StringAttribute{Computed: true, MarkdownDescription: "Account full name."},
						"email":    schema.StringAttribute{Computed: true, MarkdownDescription: "Account email address."},
					},
				},
			},
			"privileges": schema.SetAttribute{
				MarkdownDescription: "Flat list of privilege strings granted to the group.",
				Computed:            true,
				ElementType:         types.StringType,
			},
			"timeouts": timeouts.Attributes(ctx),
		},
	}
}

// Configure wires the Jamf Pro client into the data source.
func (d *AccountGroupDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	client, diags := providerdata.ConfigurePro(ctx, req.ProviderData, minJamfProVersion, "jamfplatform_pro_account_group")
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	d.client = client
}

// Read fetches an account group by id or display_name and populates state.
func (d *AccountGroupDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.client == nil {
		resp.Diagnostics.AddError("Provider not configured", "The provider client was not configured.")
		return
	}

	var data AccountGroupDataSourceModel
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

	var got *pro.AccountGroupV1
	var err error
	if !data.ID.IsNull() && data.ID.ValueString() != "" {
		got, err = d.client.GetAccountGroupV1(readCtx, data.ID.ValueString())
	} else {
		got, err = d.client.ResolveAccountGroupV1ByName(readCtx, data.DisplayName.ValueString())
	}
	if err != nil {
		resp.Diagnostics.AddError("Unable to find Jamf Pro account group", err.Error())
		return
	}

	resp.Diagnostics.Append(assignAccountGroupDataSourceModel(ctx, &data, got)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "read Jamf Pro account group data source", map[string]any{"id": data.ID.ValueString()})
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func assignAccountGroupDataSourceModel(ctx context.Context, data *AccountGroupDataSourceModel, g *pro.AccountGroupV1) diag.Diagnostics {
	var diags diag.Diagnostics
	if g == nil {
		return diags
	}
	data.ID = types.StringValue(g.ID)
	data.DisplayName = types.StringValue(g.Name)
	data.AccessLevel = types.StringValue(g.AccessLevel)
	data.PrivilegeLevel = types.StringValue(g.PrivilegeLevel)
	data.SiteID = types.StringValue(g.SiteID)
	data.LdapServerID = types.StringValue(g.LdapServerID)

	privElems := make([]attr.Value, 0, len(g.Privileges))
	for _, p := range g.Privileges {
		privElems = append(privElems, types.StringValue(p))
	}
	privSet, d := types.SetValue(types.StringType, privElems)
	diags.Append(d...)
	data.Privileges = privSet

	memberElems := make([]attr.Value, 0, len(g.Members))
	for _, m := range g.Members {
		obj, d := types.ObjectValue(memberObjectAttrTypes, map[string]attr.Value{
			"id":       types.StringValue(m.ID),
			"username": helpers.StringPointerValueOrNull(m.Username),
			"realname": helpers.StringPointerValueOrNull(m.Realname),
			"email":    helpers.StringPointerValueOrNull(m.Email),
		})
		diags.Append(d...)
		memberElems = append(memberElems, obj)
	}
	memberList, d := types.ListValue(types.ObjectType{AttrTypes: memberObjectAttrTypes}, memberElems)
	diags.Append(d...)
	data.Members = memberList
	return diags
}
