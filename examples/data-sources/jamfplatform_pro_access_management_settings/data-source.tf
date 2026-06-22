# Read the current Jamf Pro Access Management settings for Managed Apple Accounts.
data "jamfplatform_pro_access_management_settings" "current" {}

output "ade_server_uuid" {
  value = data.jamfplatform_pro_access_management_settings.current.automated_device_enrollment_server_uuid
}
