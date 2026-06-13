# Read the current Jamf Pro service-discovery ("well-known") settings. Returns one row
# per synced Apple Business / School Manager organization — useful for discovering the
# Server UUIDs to manage with jamfplatform_pro_service_discovery_enrollment.
data "jamfplatform_pro_service_discovery_enrollment" "current" {}

output "service_discovery_rows" {
  value = data.jamfplatform_pro_service_discovery_enrollment.current.well_known_setting
}
