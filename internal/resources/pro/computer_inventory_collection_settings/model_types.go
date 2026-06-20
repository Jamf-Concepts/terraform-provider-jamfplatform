// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package computer_inventory_collection_settings

import (
	datasourceTimeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/datasource/timeouts"
	resourceTimeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ComputerInventoryCollectionSettingsResourceModel represents the Terraform resource
// model for Jamf Pro computer inventory collection settings (V2).
//
// The 15 collection-preference toggles are flattened to top-level attributes (the V2
// wire nests them under computerInventoryCollectionPreferences). application_search_paths
// is the custom APPLICATION search-path collection — see state_builders.go for why
// only application-scope paths and only user-created (id != "-1") entries are modelled.
type ComputerInventoryCollectionSettingsResourceModel struct {
	ID types.String `tfsdk:"id"`

	// computerInventoryCollectionPreferences (General + Software toggles)
	CollectLocalUserAccounts                   types.Bool `tfsdk:"collect_local_user_accounts"`                      // includeAccounts
	IncludeHomeDirectorySizes                  types.Bool `tfsdk:"include_home_directory_sizes"`                     // calculateSizes
	IncludeHiddenAccounts                      types.Bool `tfsdk:"include_hidden_accounts"`                          // includeHiddenAccounts
	CollectPrinters                            types.Bool `tfsdk:"collect_printers"`                                 // includePrinters
	CollectActiveServices                      types.Bool `tfsdk:"collect_active_services"`                          // includeServices
	CollectSyncedMobileDeviceBackupDates       types.Bool `tfsdk:"collect_synced_mobile_device_backup_dates"`        // collectSyncedMobileDeviceInfo
	CollectUserAndLocationFromDirectoryService types.Bool `tfsdk:"collect_user_and_location_from_directory_service"` // updateLdapInfoOnComputerInventorySubmissions
	CollectPackageReceipts                     types.Bool `tfsdk:"collect_package_receipts"`                         // includePackages
	CollectAvailableSoftwareUpdates            types.Bool `tfsdk:"collect_available_software_updates"`               // includeSoftwareUpdates
	CollectUnmanagedCertificates               types.Bool `tfsdk:"collect_unmanaged_certificates"`                   // collectUnmanagedCertificates
	MonitorBeaconRegions                       types.Bool `tfsdk:"monitor_beacon_regions"`                           // monitorBeacons
	AllowJamfBinaryUserAndLocationChanges      types.Bool `tfsdk:"allow_jamf_binary_user_and_location_changes"`      // allowChangingUserAndLocation
	CollectApplicationUsageInformation         types.Bool `tfsdk:"collect_application_usage_information"`            // monitorApplicationUsage
	UseUnixUserPaths                           types.Bool `tfsdk:"use_unix_user_paths"`                              // useUnixUserPaths
	IncludeSoftwareID                          types.Bool `tfsdk:"include_software_id"`                              // includeSoftwareId (Computed-only)

	// applicationPaths (custom APPLICATION search paths; built-ins id == "-1" excluded)
	ApplicationSearchPaths types.Set `tfsdk:"application_search_paths"`

	Timeouts resourceTimeouts.Value `tfsdk:"timeouts"`
}

// ComputerInventoryCollectionSettingsDataSourceModel represents the Terraform data
// source model. Every attribute is Computed.
type ComputerInventoryCollectionSettingsDataSourceModel struct {
	ID types.String `tfsdk:"id"`

	CollectLocalUserAccounts                   types.Bool `tfsdk:"collect_local_user_accounts"`
	IncludeHomeDirectorySizes                  types.Bool `tfsdk:"include_home_directory_sizes"`
	IncludeHiddenAccounts                      types.Bool `tfsdk:"include_hidden_accounts"`
	CollectPrinters                            types.Bool `tfsdk:"collect_printers"`
	CollectActiveServices                      types.Bool `tfsdk:"collect_active_services"`
	CollectSyncedMobileDeviceBackupDates       types.Bool `tfsdk:"collect_synced_mobile_device_backup_dates"`
	CollectUserAndLocationFromDirectoryService types.Bool `tfsdk:"collect_user_and_location_from_directory_service"`
	CollectPackageReceipts                     types.Bool `tfsdk:"collect_package_receipts"`
	CollectAvailableSoftwareUpdates            types.Bool `tfsdk:"collect_available_software_updates"`
	CollectUnmanagedCertificates               types.Bool `tfsdk:"collect_unmanaged_certificates"`
	MonitorBeaconRegions                       types.Bool `tfsdk:"monitor_beacon_regions"`
	AllowJamfBinaryUserAndLocationChanges      types.Bool `tfsdk:"allow_jamf_binary_user_and_location_changes"`
	CollectApplicationUsageInformation         types.Bool `tfsdk:"collect_application_usage_information"`
	UseUnixUserPaths                           types.Bool `tfsdk:"use_unix_user_paths"`
	IncludeSoftwareID                          types.Bool `tfsdk:"include_software_id"`

	ApplicationSearchPaths types.Set `tfsdk:"application_search_paths"`

	Timeouts datasourceTimeouts.Value `tfsdk:"timeouts"`
}

// computerInventoryCollectionSettingsIdentityModel represents the identity object used on import.
type computerInventoryCollectionSettingsIdentityModel struct {
	ID types.String `tfsdk:"id"`
}
