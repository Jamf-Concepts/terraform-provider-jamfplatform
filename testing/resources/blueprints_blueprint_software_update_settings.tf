resource "jamfplatform_blueprints_blueprint" "test_software_update_settings" {
  name        = "Terraform Test Software Update Settings ${var.test_id}"
  description = "Managed by Terraform"

  device_groups = [data.jamfpro_group.test_target_computer_group.group_platform_id]

  software_update_settings {
    allow_standard_user_os_updates           = true
    automatic_download                       = "AlwaysOn"
    automatic_install_os_updates             = "AlwaysOn"
    automatic_install_security_updates       = "AlwaysOn"
    beta_program_enrollment                  = "Allowed"
    deferral_combined_period_days            = 7
    deferral_major_period_days               = 30
    deferral_minor_period_days               = 14
    deferral_system_period_days              = 3
    notifications_enabled                    = true
    rapid_security_response_enabled          = true
    rapid_security_response_rollback_enabled = false
    recommended_cadence                      = "Newest"

    # Beta offer programs
    beta_offer_programs {
      token       = "beta-token-1"
      description = "iOS 18 Beta Program"
    }

    beta_offer_programs {
      token       = "beta-token-2"
      description = "macOS Sequoia Beta Program"
    }
  }
}
