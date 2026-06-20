// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package app_installer_title

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// AppInstallerTitleDataSourceModel is the read-only data source model for a
// single App Installer catalog title. Every attribute is returned by Jamf Pro;
// the catalog is server-managed and not user-creatable. It carries no timeouts
// field so the identical struct can back both the singular data source and the
// plural data source's nested list elements.
type AppInstallerTitleDataSourceModel struct {
	ID                       types.String               `tfsdk:"id"`
	TitleName                types.String               `tfsdk:"title_name"`
	Publisher                types.String               `tfsdk:"publisher"`
	BundleID                 types.String               `tfsdk:"bundle_id"`
	Version                  types.String               `tfsdk:"version"`
	ShortVersion             types.String               `tfsdk:"short_version"`
	Architecture             types.String               `tfsdk:"architecture"`
	MinimumOsVersion         types.String               `tfsdk:"minimum_os_version"`
	Language                 types.String               `tfsdk:"language"`
	AvailabilityDate         types.String               `tfsdk:"availability_date"`
	IconURL                  types.String               `tfsdk:"icon_url"`
	SizeInBytes              types.Int64                `tfsdk:"size_in_bytes"`
	InstallerPackageHash     types.String               `tfsdk:"installer_package_hash"`
	InstallerPackageHashType types.String               `tfsdk:"installer_package_hash_type"`
	LaunchDaemonIncluded     types.Bool                 `tfsdk:"launch_daemon_included"`
	NotificationAvailable    types.Bool                 `tfsdk:"notification_available"`
	PackageSigningIdentity   types.String               `tfsdk:"package_signing_identity"`
	SuppressAutoUpdate       types.Bool                 `tfsdk:"suppress_auto_update"`
	OriginalMediaSources     []OriginalMediaSourceModel `tfsdk:"original_media_sources"`
}

// OriginalMediaSourceModel models one original media source entry on a title.
type OriginalMediaSourceModel struct {
	Hash     types.String `tfsdk:"hash"`
	HashType types.String `tfsdk:"hash_type"`
	URL      types.String `tfsdk:"url"`
}
