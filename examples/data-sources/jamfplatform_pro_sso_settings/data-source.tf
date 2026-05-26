data "jamfplatform_pro_sso_settings" "current" {}

output "sso_configuration_type" {
  value = data.jamfplatform_pro_sso_settings.current.configuration_type
}

output "sso_enabled" {
  value = data.jamfplatform_pro_sso_settings.current.sso_enabled
}
