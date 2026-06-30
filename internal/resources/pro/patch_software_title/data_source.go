// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package patch_software_title

import (
	"context"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/proclassic"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/datasource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/providerdata"
)

// PatchSoftwareTitleDataSource implements the Terraform data source for Jamf Pro
// patch software titles. Lookup is by id only: unlike
// jamfplatform_pro_patch_external_source (id-or-name), the classic
// /patchsoftwaretitles list response exposes no display name through the SDK and
// there is no GetByName / Resolve helper, so a name selector would require an
// N-GET scan over the full catalog. The full server view is surfaced, including
// every assigned version→package pair.
type PatchSoftwareTitleDataSource struct {
	client *proclassic.Client
}

var (
	_ datasource.DataSource              = &PatchSoftwareTitleDataSource{}
	_ datasource.DataSourceWithConfigure = &PatchSoftwareTitleDataSource{}
)

// NewPatchSoftwareTitleDataSource returns a new instance of PatchSoftwareTitleDataSource.
func NewPatchSoftwareTitleDataSource() datasource.DataSource {
	return &PatchSoftwareTitleDataSource{}
}

// Metadata sets the data source type name for the Terraform provider.
func (d *PatchSoftwareTitleDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pro_patch_software_title"
}

// Schema returns the data source schema. id is the sole selector; the remaining
// attributes are populated from the SDK response.
func (d *PatchSoftwareTitleDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Look up a Jamf Pro patch software title by ID. (The classic list endpoint exposes no display name through the SDK, so name lookup is not supported.)" +
			dataSourcePrivileges,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Patch software title ID.",
				Required:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Display name of the patch software title.",
				Computed:            true,
			},
			"name_id": schema.StringAttribute{
				MarkdownDescription: "Patch catalog key that defines the title.",
				Computed:            true,
			},
			"source_id": schema.Int64Attribute{
				MarkdownDescription: "Patch source ID this title is sourced from.",
				Computed:            true,
			},
			"category_id": schema.StringAttribute{
				MarkdownDescription: "Jamf Pro category ID.",
				Computed:            true,
			},
			"category_name": schema.StringAttribute{
				MarkdownDescription: "Category display name.",
				Computed:            true,
			},
			"site_id": schema.StringAttribute{
				MarkdownDescription: "Jamf Pro site ID.",
				Computed:            true,
			},
			"site_name": schema.StringAttribute{
				MarkdownDescription: "Site display name.",
				Computed:            true,
			},
			"web_notification": schema.BoolAttribute{
				MarkdownDescription: "Whether a Jamf Pro notification is raised for new versions.",
				Computed:            true,
			},
			"email_notification": schema.BoolAttribute{
				MarkdownDescription: "Whether an email notification is sent for new versions.",
				Computed:            true,
			},
			"version_packages": schema.MapAttribute{
				MarkdownDescription: "Every version→package assignment on the title (software_version → package ID).",
				Computed:            true,
				ElementType:         types.StringType,
			},
			"available_versions": schema.ListAttribute{
				MarkdownDescription: "All software_version strings the patch source publishes for this title.",
				Computed:            true,
				ElementType:         types.StringType,
			},
			"timeouts": timeouts.Attributes(ctx),
		},
	}
}

// Configure wires the Jamf ProClassic client into the data source via the shared
// providerdata.ConfigureProClassic helper.
func (d *PatchSoftwareTitleDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	client, diags := providerdata.ConfigureProClassic(ctx, req.ProviderData, minJamfProVersion, "jamfplatform_pro_patch_software_title")
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	d.client = client
}

// Read fetches a patch software title by ID and populates Terraform state.
func (d *PatchSoftwareTitleDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.client == nil {
		resp.Diagnostics.AddError(
			"Provider not configured",
			"The provider client was not configured. Please ensure the provider block is set up correctly.",
		)
		return
	}

	var data PatchSoftwareTitleDataSourceModel
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

	if data.ID.IsNull() || data.ID.ValueString() == "" {
		resp.Diagnostics.AddError("Missing patch software title selector", "id must be supplied.")
		return
	}

	got, err := d.client.GetPatchSoftwareTitleByID(readCtx, data.ID.ValueString()) //nolint:staticcheck // SA1019: classic /patchsoftwaretitles intentionally used; v2 create unusable — see crud.go header note
	if err != nil {
		resp.Diagnostics.AddError("Unable to find Jamf Pro patch software title", err.Error())
		return
	}
	resp.Diagnostics.Append(assignPatchSoftwareTitleDataSourceModel(readCtx, &data, got)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "read Jamf Pro patch software title data source", map[string]any{"id": data.ID.ValueString()})
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
