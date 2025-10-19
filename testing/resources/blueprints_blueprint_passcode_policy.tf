resource "jamfplatform_blueprints_blueprint" "test_passcode_policy" {
  name        = "Terraform Test Passcode Policy ${var.test_id}"
  description = "Managed by Terraform"

  device_groups = [data.jamfpro_group.test_target_computer_group.group_platform_id,
  data.jamfpro_group.test_target_mobile_device_group.group_platform_id]

  passcode_policy {
    change_at_next_auth              = true
    failed_attempts_reset_in_minutes = 0
    maximum_failed_attempts          = 11
    maximum_grace_period_in_minutes  = 0
    maximum_inactivity_in_minutes    = 0
    maximum_passcode_age_in_days     = 0
    minimum_complex_characters       = 0
    minimum_length                   = 0
    passcode_reuse_limit             = 1
    require_alphanumeric_passcode    = true
    require_complex_passcode         = true
    require_passcode                 = true
  }
}
