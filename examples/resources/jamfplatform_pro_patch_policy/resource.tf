# A patch policy deploys a single target_version of a patch software title to a
# scoped set of computers. It is created against a software_title_configuration
# (a jamfplatform_pro_patch_software_title ID), and the target_version must be a
# version that has a package assigned on that title.

resource "jamfplatform_pro_package" "work_8_33" {
  display_name = "8x8 Work 8.33.2.2"
  file_name    = "8x8-work-8.33.2.2.pkg"
}

resource "jamfplatform_pro_patch_software_title" "eight_by_eight" {
  name      = "8x8 Work"
  name_id   = "285"
  source_id = 1

  # The policy can only target a version that has a package assigned here.
  version_packages = {
    "8.33.2.2" = jamfplatform_pro_package.work_8_33.id
  }
}

resource "jamfplatform_pro_building" "hq" {
  name = "HQ"
}

resource "jamfplatform_pro_patch_policy" "work_latest" {
  software_title_configuration_id = jamfplatform_pro_patch_software_title.eight_by_eight.id
  name                            = "8x8 Work - 8.33.2.2"
  target_version                  = "8.33.2.2"

  enabled             = true
  distribution_method = "selfservice" # "Make Available in Self Service"; use "prompt" to install automatically.
  allow_downgrade     = false
  patch_unknown       = false

  # Scope is computer-only (no users). A policy can only be enabled when its
  # scope resolves to at least one in-site smart group. "1" is the default
  # "All Managed Clients" smart group.
  scope = {
    computer_group_ids = ["1"]
    building_ids       = [jamfplatform_pro_building.hq.id]

    limitations = {
      network_segment_ids = []
      ibeacon_ids         = []
    }

    exclusions = {
      computer_group_ids = []
    }
  }

  user_interaction = {
    install_button_text      = "Update Now"
    self_service_description = "Update 8x8 Work to the approved version."

    notifications = {
      enabled = true
      subject = "8x8 Work update available"
      reminders = {
        enabled   = true
        frequency = 24
      }
    }

    deadlines = {
      enabled = true
      period  = 7
    }

    grace_period = {
      duration                    = 15
      notification_center_subject = "Important"
    }
  }
}

# Server-derived from the target version's patch definition (read-only).
output "patch_policy_kill_apps" {
  value = jamfplatform_pro_patch_policy.work_latest.kill_apps
}
