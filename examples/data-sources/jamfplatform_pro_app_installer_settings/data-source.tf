data "jamfplatform_pro_app_installer_settings" "current" {}

output "deployment_batch_size" {
  value = data.jamfplatform_pro_app_installer_settings.current.deployment_settings != null ? data.jamfplatform_pro_app_installer_settings.current.deployment_settings.batch_size : null
}
