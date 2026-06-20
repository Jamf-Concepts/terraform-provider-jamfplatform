// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

// Package app_installer_title implements the read-only
// jamfplatform_pro_app_installer_title data source, which looks up a single
// App Installer catalog title by ID. The App Installer catalog is managed by
// Jamf Pro; titles are not user-creatable. Use this data source to discover a
// title's ID for jamfplatform_pro_app_installer.app_title_id.
package app_installer_title

import (
	"context"
	"time"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/pro"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/providerdata"
)

const defaultReadTimeout = 60 * time.Second

// minJamfProVersion is the minimum Jamf Pro tenant version required by this data
// source. Empty: the App Installer endpoints predate the provider's overall
// floor (the App Catalog shipped well before the 11.0.0 minimum).
const minJamfProVersion = ""

// AppInstallerTitleDataSource implements the Terraform data source for a single
// App Installer catalog title.
type AppInstallerTitleDataSource struct {
	client *pro.Client
}

var _ datasource.DataSource = &AppInstallerTitleDataSource{}

// NewAppInstallerTitleDataSource returns a new instance of AppInstallerTitleDataSource.
func NewAppInstallerTitleDataSource() datasource.DataSource {
	return &AppInstallerTitleDataSource{}
}

// Metadata sets the data source type name for the Terraform provider.
func (d *AppInstallerTitleDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pro_app_installer_title"
}

// Schema returns the data source schema.
func (d *AppInstallerTitleDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	attrs := TitleDataSourceAttributes()
	attrs["id"] = schema.StringAttribute{
		MarkdownDescription: "Catalog title ID to look up.",
		Required:            true,
	}
	resp.Schema = schema.Schema{
		MarkdownDescription: "Looks up a single App Installer catalog title by ID. Titles are published by Jamf and cannot be created or modified; this data source surfaces a title's metadata so you can reference its `id` from `jamfplatform_pro_app_installer.app_title_id`.",
		Attributes:          attrs,
	}
}

// Configure wires the Jamf Pro client into the data source.
func (d *AppInstallerTitleDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	client, diags := providerdata.ConfigurePro(ctx, req.ProviderData, minJamfProVersion, "jamfplatform_pro_app_installer_title")
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	d.client = client
}

// Read fetches a title by ID and populates Terraform state.
func (d *AppInstallerTitleDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.client == nil {
		resp.Diagnostics.AddError(
			"Provider not configured",
			"The provider client was not configured. Please ensure the provider block is set up correctly.",
		)
		return
	}

	var data AppInstallerTitleDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.ID.IsNull() || data.ID.ValueString() == "" {
		resp.Diagnostics.AddError("Missing ID", "The id attribute must be provided to read an App Installer title.")
		return
	}

	readCtx, cancel := context.WithTimeout(ctx, defaultReadTimeout)
	defer cancel()

	got, err := d.client.GetAppInstallerTitleV1(readCtx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to find App Installer title", err.Error())
		return
	}

	data = AssignTitleDataSource(got)

	tflog.Trace(ctx, "read App Installer title data source", map[string]any{"id": data.ID.ValueString()})
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// TitleDataSourceAttributes returns the shared Computed attribute map describing
// an App Installer catalog title. Shared with the plural titles data source so
// both surface an identical field set. The caller adds the lookup `id` shape and
// any `timeouts` attribute appropriate to its construct.
func TitleDataSourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id":                          computedString("Catalog title ID."),
		"title_name":                  computedString("Title display name."),
		"publisher":                   computedString("Title publisher."),
		"bundle_id":                   computedString("Primary application bundle identifier."),
		"version":                     computedString("Current published version."),
		"short_version":               computedString("Current published short version string."),
		"architecture":                computedString("Supported CPU architecture (e.g. `x86_64`, `arm64`, `universal`)."),
		"minimum_os_version":          computedString("Minimum macOS version required to install the title."),
		"language":                    computedString("Title language."),
		"availability_date":           computedString("Date the current version became available in the catalog."),
		"icon_url":                    computedString("URL of the title icon."),
		"size_in_bytes":               schema.Int64Attribute{MarkdownDescription: "Installer package size in bytes.", Computed: true},
		"installer_package_hash":      computedString("Hash of the installer package."),
		"installer_package_hash_type": computedString("Algorithm used for the installer package hash (e.g. `SHA_256`)."),
		"launch_daemon_included":      schema.BoolAttribute{MarkdownDescription: "Whether the title bundles a launch daemon.", Computed: true},
		"notification_available":      schema.BoolAttribute{MarkdownDescription: "Whether the title supports end-user install notifications.", Computed: true},
		"package_signing_identity":    computedString("Signing identity of the installer package."),
		"suppress_auto_update":        schema.BoolAttribute{MarkdownDescription: "Whether the title suppresses its built-in auto-update mechanism when managed by Jamf.", Computed: true},
		"original_media_sources": schema.ListNestedAttribute{
			MarkdownDescription: "Original media sources Jamf used to build the title's installer package.",
			Computed:            true,
			NestedObject: schema.NestedAttributeObject{
				Attributes: map[string]schema.Attribute{
					"hash":      computedString("Media source hash."),
					"hash_type": computedString("Media source hash algorithm."),
					"url":       computedString("Media source URL."),
				},
			},
		},
	}
}

// AssignTitleDataSource maps an SDK title into the data source model.
func AssignTitleDataSource(t *pro.AppInstallerTitle) AppInstallerTitleDataSourceModel {
	out := AppInstallerTitleDataSourceModel{}
	if t == nil {
		return out
	}
	out.ID = types.StringValue(t.ID)
	out.TitleName = types.StringValue(t.TitleName)
	out.Publisher = types.StringValue(t.Publisher)
	out.BundleID = types.StringValue(t.BundleID)
	out.Version = types.StringValue(t.Version)
	out.ShortVersion = types.StringValue(t.ShortVersion)
	out.Architecture = types.StringValue(t.Architecture)
	out.MinimumOsVersion = types.StringValue(t.MinimumOsVersion)
	out.Language = types.StringValue(t.Language)
	out.AvailabilityDate = types.StringValue(t.AvailabilityDate)
	out.IconURL = types.StringValue(t.IconURL)
	out.SizeInBytes = types.Int64Value(int64(t.SizeInBytes))
	out.InstallerPackageHash = types.StringValue(t.InstallerPackageHash)
	out.InstallerPackageHashType = types.StringValue(t.InstallerPackageHashType)
	out.LaunchDaemonIncluded = types.BoolValue(t.LaunchDaemonIncluded)
	out.NotificationAvailable = types.BoolValue(t.NotificationAvailable)
	out.PackageSigningIdentity = types.StringValue(t.PackageSigningIdentity)
	out.SuppressAutoUpdate = types.BoolValue(t.SuppressAutoUpdate)
	out.OriginalMediaSources = make([]OriginalMediaSourceModel, 0, len(t.OriginalMediaSources))
	for _, m := range t.OriginalMediaSources {
		out.OriginalMediaSources = append(out.OriginalMediaSources, OriginalMediaSourceModel{
			Hash:     types.StringValue(m.Hash),
			HashType: types.StringValue(m.HashType),
			URL:      types.StringValue(m.URL),
		})
	}
	return out
}

// computedString returns a Computed-only StringAttribute for a server-derived
// title field.
func computedString(desc string) schema.StringAttribute {
	return schema.StringAttribute{MarkdownDescription: desc, Computed: true}
}
