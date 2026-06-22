# Read the current Jamf Pro login page disclaimer settings.
data "jamfplatform_pro_login_page_settings" "current" {}

output "include_custom_disclaimer" {
  value = data.jamfplatform_pro_login_page_settings.current.include_custom_disclaimer
}
