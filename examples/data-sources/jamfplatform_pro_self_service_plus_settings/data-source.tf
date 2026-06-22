data "jamfplatform_pro_self_service_plus_settings" "current" {}

output "self_service_plus_enabled" {
  value = data.jamfplatform_pro_self_service_plus_settings.current.enabled
}
