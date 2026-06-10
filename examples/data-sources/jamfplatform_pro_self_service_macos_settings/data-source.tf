# Read the current Self Service for macOS app settings.
data "jamfplatform_pro_self_service_macos_settings" "current" {}

output "self_service_login_method" {
  value = data.jamfplatform_pro_self_service_macos_settings.current.login_method
}
