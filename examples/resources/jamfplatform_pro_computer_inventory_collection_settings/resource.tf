resource "jamfplatform_pro_computer_inventory_collection_settings" "this" {
  # Inventory Collection (General)
  collect_local_user_accounts                      = true
  include_home_directory_sizes                     = false
  include_hidden_accounts                          = false
  collect_printers                                 = true
  collect_active_services                          = true
  collect_synced_mobile_device_backup_dates        = false
  collect_user_and_location_from_directory_service = true
  collect_package_receipts                         = true
  collect_available_software_updates               = false
  collect_unmanaged_certificates                   = true
  monitor_beacon_regions                           = false
  allow_jamf_binary_user_and_location_changes      = true

  # Software → Applications
  collect_application_usage_information = false
  use_unix_user_paths                   = true

  # Custom application search paths (built-in paths are managed by Jamf Pro).
  # The V2 API supports application-scope custom paths only (Fonts/Plug-ins are not exposed).
  application_search_paths = [
    "/Library/MyOrg/Applications/",
  ]
}
