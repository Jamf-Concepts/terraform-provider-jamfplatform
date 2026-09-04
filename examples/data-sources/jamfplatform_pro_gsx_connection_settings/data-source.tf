# Read the current Jamf Pro GSX Connection settings. The token and keystore
# secrets are never returned by Jamf Pro and are not exposed here.
data "jamfplatform_pro_gsx_connection_settings" "current" {}

output "gsx_enabled" {
  value = data.jamfplatform_pro_gsx_connection_settings.current.enabled
}

output "gsx_certificate_expiration_epoch" {
  value = data.jamfplatform_pro_gsx_connection_settings.current.keystore_expiration_epoch
}
