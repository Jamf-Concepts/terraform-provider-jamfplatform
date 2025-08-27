# Software Update Settings Blueprint
resource "jamfplatform_blueprints_blueprint" "software_update" {
  name = "Software Update Settings Blueprint"

  # Use the same device group as the security blueprint
  device_groups = [
    "fce3d9a5-8660-42ff-a95e-625e7b53b48a"
  ]

  # Blueprint component
  component {
    identifier = "com.jamf.ddm.software-update-settings"
    configuration = {
      AllowStandardUserOSUpdates_Enabled              = true
      AllowStandardUserOSUpdates_Included             = true
      AutomaticActions_Download_Included              = true
      AutomaticActions_Download_Value                 = "AlwaysOn"
      AutomaticActions_InstallOSUpdates_Included      = true
      AutomaticActions_InstallOSUpdates_Value         = "AlwaysOn"
      AutomaticActions_InstallSecurityUpdate_Included = true
      AutomaticActions_InstallSecurityUpdate_Value    = "AlwaysOn"
      Beta_Included                                   = true
      Beta_Value_ProgramEnrollment                    = "AlwaysOff"
      Deferrals_CombinedPeriodInDays_Included         = true
      Deferrals_CombinedPeriodInDays_Value            = 7
      Deferrals_MajorPeriodInDays_Included            = true
      Deferrals_MajorPeriodInDays_Value               = 30
      Deferrals_MinorPeriodInDays_Included            = true
      Deferrals_MinorPeriodInDays_Value               = 7
      Deferrals_SystemPeriodInDays_Included           = true
      Deferrals_SystemPeriodInDays_Value              = 7
      Notifications_Enabled                           = true
      Notifications_Included                          = true
      RapidSecurityResponse_EnableRollback_Enabled    = false
      RapidSecurityResponse_EnableRollback_Included   = true
      RapidSecurityResponse_Enable_Enabled            = true
      RapidSecurityResponse_Enable_Included           = true
      RecommendedCadence_Included                     = true
      RecommendedCadence_Value                        = "Newest"
    }
  }
}

# Security Settings Blueprint (Disk Management and Passcode)
resource "jamfplatform_blueprints_blueprint" "security" {
  provider    = jamfplatform.blueprints
  name        = "Security Blueprint"
  description = "Created by Terraform"

  # Device groups to target (must be valid device group IDs)
  device_groups = [
    "fce3d9a5-8660-42ff-a95e-625e7b53b48a"
  ]

  # Blueprint components
  component {
    identifier = "com.jamf.ddm.disk-management"
    configuration = {
      version                               = 2
      Restrictions_NetworkStorage_Value     = "ReadOnly"
      Restrictions_NetworkStorage_Included  = true
      Restrictions_ExternalStorage_Value    = "Disallowed"
      Restrictions_ExternalStorage_Included = true
    }
  }

  component {
    identifier = "com.jamf.ddm.passcode-settings"
    configuration = {
      RequirePasscode             = true
      ChangeAtNextAuth            = false
      RequireComplexPasscode      = true
      MinimumComplexCharacters    = 1
      RequireAlphanumericPasscode = false
    }
  }
}

# Safari Settings Blueprint
resource "jamfplatform_blueprints_blueprint" "safari_settings" {
  provider    = jamfplatform.blueprints
  name        = "Safari Settings Blueprint"
  description = "Created by Terraform"

  # Use the same device group as the security blueprint
  device_groups = [
    "fce3d9a5-8660-42ff-a95e-625e7b53b48a"
  ]

  # Blueprint component
  component {
    identifier = "com.jamf.ddm.safari-settings"
    configuration = {
      AllowHistoryClearing_Value          = false
      AllowHistoryClearing_Included       = true
      AllowPrivateBrowsing_Value          = false
      AllowPrivateBrowsing_Included       = true
      AllowDisablingFraudWarning_Value    = false
      AllowDisablingFraudWarning_Included = true
    }
  }
}
