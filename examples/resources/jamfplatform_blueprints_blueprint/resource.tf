# Software Update Settings Blueprint
resource "jamfplatform_blueprints_blueprint" "software_update_settings" {
  name        = "Software Update Settings"
  description = "Managed by Terraform"

  device_groups = ["fce3d9a5-8660-42ff-a95e-625e7b53b48a"]

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

# Latest OS version Software Updates Blueprint
resource "jamfplatform_blueprints_blueprint" "automatic_software_updates" {
  name        = "Latest OS version Software Updates"
  description = "Managed by Terraform"

  device_groups = ["fce3d9a5-8660-42ff-a95e-625e7b53b48a"]

  software_update {
    deployment_time    = "02:00"
    enforce_after_days = 7
  }
}

# Specific OS version and time Software Updates Blueprint
resource "jamfplatform_blueprints_blueprint" "manual_software_updates" {
  name        = "Specific OS Version and time Software Updates"
  description = "Managed by Terraform"

  device_groups = ["fce3d9a5-8660-42ff-a95e-625e7b53b48a"]

  software_update {
    target_os_version      = "26.0.1"
    target_local_date_time = "2025-10-10T12:00:00"
  }
}

# Legacy Payloads Example Blueprint
resource "jamfplatform_blueprints_blueprint" "legacy_payloads_example" {
  name        = "Restrictions for Safari"
  description = "Managed by Terraform"

  device_groups = ["fce3d9a5-8660-42ff-a95e-625e7b53b48a"]

  legacy_payloads = jsonencode([
    {
      allowSafariHistoryClearing = false
      allowSafariPrivateBrowsing = false
      payloadType                = "com.apple.applicationaccess"
      payloadIdentifier          = "da0ac44c-419e-43ff-b300-00b0e645fa7e"
    }
  ])
}
