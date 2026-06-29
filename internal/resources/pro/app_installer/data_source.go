// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package app_installer

import (
	"context"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/pro"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/datasource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/datasourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/providerdata"
)

// AppInstallerDataSource implements the Terraform data source for a single App
// Installer deployment. Lookup is by ID or by exact name — exactly one must be
// supplied. It surfaces a flat projection of the deployment's scalar fields; for
// the nested presentation blocks, manage the deployment as a resource.
type AppInstallerDataSource struct {
	client *pro.Client
}

var (
	_ datasource.DataSource                     = &AppInstallerDataSource{}
	_ datasource.DataSourceWithConfigure        = &AppInstallerDataSource{}
	_ datasource.DataSourceWithConfigValidators = &AppInstallerDataSource{}
)

// NewAppInstallerDataSource returns a new instance of AppInstallerDataSource.
func NewAppInstallerDataSource() datasource.DataSource {
	return &AppInstallerDataSource{}
}

// Metadata sets the data source type name.
func (d *AppInstallerDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pro_app_installer"
}

// Schema returns the data source schema.
func (d *AppInstallerDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Look up a single App Installer deployment by ID or by exact name. Exactly one of `id` or `name` must be supplied. Surfaces a flat read-only projection; manage the deployment as a resource for the nested presentation blocks." + dataSourcePrivileges,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Deployment ID. Mutually exclusive with `name`.",
				Optional:            true,
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Deployment display name (exact match). Mutually exclusive with `id`.",
				Optional:            true,
				Computed:            true,
			},
			"enabled":                            schema.BoolAttribute{MarkdownDescription: "Whether the deployment is enabled.", Computed: true},
			"app_title_name":                     schema.StringAttribute{MarkdownDescription: "Name of the App Catalog title being deployed.", Computed: true},
			"app_title_id":                       schema.StringAttribute{MarkdownDescription: "App Catalog title ID being deployed.", Computed: true},
			"deployment_type":                    schema.StringAttribute{MarkdownDescription: "Delivery method.", Computed: true},
			"update_behavior":                    schema.StringAttribute{MarkdownDescription: "Update behavior.", Computed: true},
			"selected_version":                   schema.StringAttribute{MarkdownDescription: "Version Jamf Pro has selected. Empty under `AUTOMATIC`.", Computed: true},
			"latest_available_version":           schema.StringAttribute{MarkdownDescription: "Latest version available in the catalog.", Computed: true},
			"title_available_in_ais":             schema.BoolAttribute{MarkdownDescription: "Whether the title is available in the App Installers catalog.", Computed: true},
			"version_removed":                    schema.BoolAttribute{MarkdownDescription: "Whether the selected version has been removed from the catalog.", Computed: true},
			"category_id":                        schema.StringAttribute{MarkdownDescription: "Category ID. `-1` means none.", Computed: true},
			"site_id":                            schema.StringAttribute{MarkdownDescription: "Site ID. `-1` means none.", Computed: true},
			"smart_group_id":                     schema.StringAttribute{MarkdownDescription: "Smart computer group ID. `-1` means none.", Computed: true},
			"install_predefined_config_profiles": schema.BoolAttribute{MarkdownDescription: "Whether predefined configuration profiles are installed.", Computed: true},
			"trigger_admin_notifications":        schema.BoolAttribute{MarkdownDescription: "Whether administrator notifications are raised.", Computed: true},
			"timeouts":                           timeouts.Attributes(ctx),
		},
	}
}

// ConfigValidators enforces that exactly one of id / name is supplied.
func (d *AppInstallerDataSource) ConfigValidators(ctx context.Context) []datasource.ConfigValidator {
	return []datasource.ConfigValidator{
		datasourcevalidator.ExactlyOneOf(
			path.MatchRoot("id"),
			path.MatchRoot("name"),
		),
	}
}

// Configure wires the Jamf Pro client into the data source.
func (d *AppInstallerDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	client, diags := providerdata.ConfigurePro(ctx, req.ProviderData, minJamfProVersion, "jamfplatform_pro_app_installer")
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	d.client = client
}

// Read fetches a deployment by ID or by name and populates Terraform state.
func (d *AppInstallerDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.client == nil {
		resp.Diagnostics.AddError(
			"Provider not configured",
			"The provider client was not configured. Please ensure the provider block is set up correctly.",
		)
		return
	}

	var data AppInstallerDataSourceModel
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

	var (
		got *pro.AppInstallerDeployment
		err error
	)
	switch {
	case !data.ID.IsNull() && data.ID.ValueString() != "":
		got, err = d.client.GetAppInstallerDeploymentV1(readCtx, data.ID.ValueString())
	case !data.Name.IsNull() && data.Name.ValueString() != "":
		// Resolve the ID by name, then GET the flat deployment. The SDK's
		// ResolveAppInstallerDeploymentV1ByName decodes the matched *list*
		// element, which is the expanded shape (app/site/category/smartGroup
		// nested refs) — unmarshaling it into the flat AppInstallerDeployment
		// silently drops ~10 scalar fields (app_title_id, category_id, etc.).
		// Resolve-id-then-GET hits the canonical flat representation instead.
		var id string
		id, err = d.client.ResolveAppInstallerDeploymentV1IDByName(readCtx, data.Name.ValueString())
		if err == nil {
			got, err = d.client.GetAppInstallerDeploymentV1(readCtx, id)
		}
	default:
		resp.Diagnostics.AddError("Missing deployment selector", "Exactly one of id or name must be supplied.")
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Unable to find App Installer deployment", err.Error())
		return
	}
	assignAppInstallerDataSourceModel(&data, got)

	// Reverse-resolve app_title_id → app_title_name (the deployment GET returns
	// only the ID). Best-effort: leave app_title_name null on a catalog hiccup.
	if name, ok := titleNameForID(readCtx, d.client, data.AppTitleID.ValueString()); ok {
		data.AppTitleName = types.StringValue(name)
	}

	tflog.Trace(ctx, "read App Installer deployment data source", map[string]any{"id": data.ID.ValueString(), "name": data.Name.ValueString()})
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
