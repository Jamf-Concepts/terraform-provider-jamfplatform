data "jamfplatform_pro_user_initiated_enrollment_settings" "current" {}

output "computer_enrollment_enabled" {
  value = data.jamfplatform_pro_user_initiated_enrollment_settings.current.enable_computer_enrollment
}

output "access_groups" {
  value = data.jamfplatform_pro_user_initiated_enrollment_settings.current.access_group
}
