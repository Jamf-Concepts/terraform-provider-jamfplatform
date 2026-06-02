// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package patch_software_title

import (
	datasourceTimeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/datasource/timeouts"
	resourceTimeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/filters"
)

// PatchSoftwareTitleResourceModel represents the Terraform resource model for a
// Jamf Pro patch software title.
//
// VersionPackages is a managed-subset map keyed by software_version with
// package_id string values. The user only declares the version→package
// assignments they care about; the server holds the full catalog of versions
// (20+) for the title and merges per software_version on write. Read therefore
// reconciles only the keys the user declared (see state_builders.go).
type PatchSoftwareTitleResourceModel struct {
	ID                        types.String                `tfsdk:"id"`
	Name                      types.String                `tfsdk:"name"`
	NameID                    types.String                `tfsdk:"name_id"`
	SourceID                  types.Int64                 `tfsdk:"source_id"`
	CategoryID                types.String                `tfsdk:"category_id"`
	CategoryName              types.String                `tfsdk:"category_name"`
	SiteID                    types.String                `tfsdk:"site_id"`
	SiteName                  types.String                `tfsdk:"site_name"`
	WebNotification           types.Bool                  `tfsdk:"web_notification"`
	EmailNotification         types.Bool                  `tfsdk:"email_notification"`
	VersionPackages           types.Map                   `tfsdk:"version_packages"`
	AvailableVersions         types.List                  `tfsdk:"available_versions"`
	AcceptExtensionAttributes types.Bool                  `tfsdk:"accept_extension_attributes"`
	ExtensionAttributes       []PatchSoftwareTitleEAModel `tfsdk:"extension_attributes"`
	Timeouts                  resourceTimeouts.Value      `tfsdk:"timeouts"`
}

// PatchSoftwareTitleEAModel is one Jamf-defined extension attribute attached to
// a patch software title. For some titles, Jamf creates an EA that collects the
// installed version on managed computers; it must be accepted before inventory
// is gathered. The list is read-only state (the catalog of EAs and their accept
// status); set accept_extension_attributes to accept any pending ones.
type PatchSoftwareTitleEAModel struct {
	EaID        types.String `tfsdk:"ea_id"`
	DisplayName types.String `tfsdk:"display_name"`
	Accepted    types.Bool   `tfsdk:"accepted"`
}

// PatchSoftwareTitleDataSourceModel represents the Terraform data source model.
// Lookup is by id only: the classic /patchsoftwaretitles list response surfaces
// no display name through the SDK, and there is no GetByName / Resolve helper,
// so a name selector would require an N-GET scan. See data_source.go.
type PatchSoftwareTitleDataSourceModel struct {
	ID                types.String             `tfsdk:"id"`
	Name              types.String             `tfsdk:"name"`
	NameID            types.String             `tfsdk:"name_id"`
	SourceID          types.Int64              `tfsdk:"source_id"`
	CategoryID        types.String             `tfsdk:"category_id"`
	CategoryName      types.String             `tfsdk:"category_name"`
	SiteID            types.String             `tfsdk:"site_id"`
	SiteName          types.String             `tfsdk:"site_name"`
	WebNotification   types.Bool               `tfsdk:"web_notification"`
	EmailNotification types.Bool               `tfsdk:"email_notification"`
	VersionPackages   types.Map                `tfsdk:"version_packages"`
	AvailableVersions types.List               `tfsdk:"available_versions"`
	Timeouts          datasourceTimeouts.Value `tfsdk:"timeouts"`
}

// patchSoftwareTitleIdentityModel represents the identity object for the
// resource and list results.
type patchSoftwareTitleIdentityModel struct {
	ID types.String `tfsdk:"id"`
}

// PatchSoftwareTitleListResourceModel represents the config model for list
// queries. Classic has no RSQL — the filter shape is the shared client-side
// substring block. NOTE: the SDK list item exposes no display name, so the
// filter matches name_id (the catalog key), not the display name.
type PatchSoftwareTitleListResourceModel struct {
	Filter *filters.ClassicFilterModel `tfsdk:"filter"`
}
