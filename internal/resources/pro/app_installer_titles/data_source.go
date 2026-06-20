// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

// Package app_installer_titles implements the read-only
// jamfplatform_pro_app_installer_titles data source, which returns the full
// App Installer catalog (every published title) for the tenant. The catalog is
// server-managed; titles are not user-creatable. Use it to discover the `id` of
// a title to reference from jamfplatform_pro_app_installer.app_title_id.
package app_installer_titles

import (
	"context"
	"strings"
	"time"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/pro"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/providerdata"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/app_installer_title"
)

const defaultReadTimeout = 90 * time.Second

// minJamfProVersion is the minimum Jamf Pro tenant version required by this data
// source. Empty: the App Installer endpoints predate the provider's overall
// floor.
const minJamfProVersion = ""

// AppInstallerTitlesDataSourceModel is the plural data source model. `titles`
// carries the full catalog; the optional `name_substring` narrows the result
// client-side (the catalog endpoint has no server-side filter).
type AppInstallerTitlesDataSourceModel struct {
	ID            types.String                                           `tfsdk:"id"`
	NameSubstring types.String                                           `tfsdk:"name_substring"`
	Titles        []app_installer_title.AppInstallerTitleDataSourceModel `tfsdk:"titles"`
}

// AppInstallerTitlesDataSource implements the Terraform plural data source.
type AppInstallerTitlesDataSource struct {
	client *pro.Client
}

var _ datasource.DataSource = &AppInstallerTitlesDataSource{}

// NewAppInstallerTitlesDataSource returns a new instance of AppInstallerTitlesDataSource.
func NewAppInstallerTitlesDataSource() datasource.DataSource {
	return &AppInstallerTitlesDataSource{}
}

// Metadata sets the data source type name for the Terraform provider.
func (d *AppInstallerTitlesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pro_app_installer_titles"
}

// Schema returns the data source schema.
func (d *AppInstallerTitlesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	titleAttrs := app_installer_title.TitleDataSourceAttributes()
	resp.Schema = schema.Schema{
		MarkdownDescription: "Returns the App Installer catalog — every title published to the tenant. Titles are managed by Jamf and cannot be created or modified. Use the optional `name_substring` to narrow the result; matching is case-insensitive and applied after the full catalog is fetched.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Internal identifier for this data source read.",
				Computed:            true,
			},
			"name_substring": schema.StringAttribute{
				MarkdownDescription: "Optional case-insensitive substring matched against each title's name. When omitted, the whole catalog is returned.",
				Optional:            true,
			},
			"titles": schema.ListNestedAttribute{
				MarkdownDescription: "Catalog titles, optionally narrowed by `name_substring`.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: titleAttrs,
				},
			},
		},
	}
}

// Configure wires the Jamf Pro client into the data source.
func (d *AppInstallerTitlesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	client, diags := providerdata.ConfigurePro(ctx, req.ProviderData, minJamfProVersion, "jamfplatform_pro_app_installer_titles")
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	d.client = client
}

// Read fetches the full catalog (optionally narrowed) and populates state.
func (d *AppInstallerTitlesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.client == nil {
		resp.Diagnostics.AddError(
			"Provider not configured",
			"The provider client was not configured. Please ensure the provider block is set up correctly.",
		)
		return
	}

	var data AppInstallerTitlesDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	readCtx, cancel := context.WithTimeout(ctx, defaultReadTimeout)
	defer cancel()

	titles, err := d.client.ListAppInstallerTitlesV1(readCtx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to list App Installer titles", err.Error())
		return
	}

	data.Titles = FilterAndMapTitles(titles, data.NameSubstring)
	data.ID = types.StringValue("app_installer_titles")

	tflog.Trace(ctx, "read App Installer titles data source", map[string]any{"returned": len(data.Titles)})
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// FilterAndMapTitles maps every SDK title into the data source model, keeping
// only titles whose name contains nameSubstring (case-insensitive) when that
// filter is set. The slice is always non-nil so an empty result serialises as
// an empty list, not null.
func FilterAndMapTitles(titles []pro.AppInstallerTitle, nameSubstring types.String) []app_installer_title.AppInstallerTitleDataSourceModel {
	out := make([]app_installer_title.AppInstallerTitleDataSourceModel, 0, len(titles))
	substr := ""
	if helpers.IsConfiguredValue(nameSubstring) {
		substr = strings.ToLower(nameSubstring.ValueString())
	}
	for i := range titles {
		if substr != "" && !strings.Contains(strings.ToLower(titles[i].TitleName), substr) {
			continue
		}
		out = append(out, app_installer_title.AssignTitleDataSource(&titles[i]))
	}
	return out
}
