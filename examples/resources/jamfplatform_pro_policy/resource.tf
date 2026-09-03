# Minimal policy: an enabled flag and a name. Every other section is optional.
resource "jamfplatform_pro_policy" "minimal" {
  general = {
    name    = "tf-acc-minimal-policy"
    enabled = true
  }
}

# Fully-scoped policy demonstrating per-target IDs sourced from sibling
# resources. Interpolate `.jamf_pro_id` on Platform Services device groups,
# because `scope.computer_group_ids` takes Jamf Pro IDs as strings.
resource "jamfplatform_pro_policy" "scoped" {
  general = {
    name            = "tf-acc-scoped-policy"
    enabled         = true
    frequency       = "Once per computer"
    trigger_checkin = true
  }

  scope = {
    targets = {
      computer_group_ids = [
        jamfplatform_device_group.engineering.jamf_pro_id,
      ]
      building_ids = [
        jamfplatform_pro_building.hq.id,
      ]
      department_ids = [
        jamfplatform_pro_department.it.id,
      ]
    }

    limitations = {
      directory_service_user_group_names = ["wheel"]
    }

    exclusions = {
      computer_ids = ["28"]
    }
  }

  self_service = {
    use_for_self_service      = true
    self_service_display_name = "tf-acc Demo"
    display_notifications     = true
    notification_location     = "Self Service"
    notification_subject      = "tf-acc"
    notification_message      = "Demo policy now available."

    # A policy can appear in multiple Self Service categories, each with its
    # own "Display in" / "Feature in" flags.
    categories = [
      {
        id         = "1"
        display_in = true
        feature_in = true
      },
      {
        id         = "2"
        display_in = true
        feature_in = false
      },
    ]
  }
}

# `all_computers = true` excludes per-target lists. The provider's
# scope.AllFlagConflictsWith validator catches inconsistent configs at plan time.
resource "jamfplatform_pro_policy" "universal" {
  general = {
    name = "tf-acc-universal-policy"
  }
  scope = {
    targets = {
      all_computers = true
    }
  }
}

# The Options sidebar payloads map to top-level blocks that mirror the admin
# UI section names. `account_maintenance` is flattened into four peer blocks:
# `local_accounts`, `management_account`, `directory_bindings`, `efi_password`.
resource "jamfplatform_pro_policy" "options" {
  general = {
    name      = "tf-acc-options-policy"
    enabled   = true
    frequency = "Once per computer"
  }

  packages = {
    distribution_point = "Cloud Distribution Point"
    packages = [
      { id = "100", action = "Install" },
    ]
  }

  # Options ▸ Local Accounts. `password` is WriteOnly. Rotate it by bumping
  # `password_wo_version`.
  local_accounts = [
    {
      action              = "Create"
      username            = "tf-admin"
      realname            = "TF Admin"
      password            = "Sup3rS3cret!"
      password_wo_version = 1
      admin               = true
    },
  ]

  # Options ▸ Management Accounts.
  management_account = {
    action                      = "rotate"
    managed_password_length     = 16
    managed_password_wo_version = 1
  }

  # Options ▸ Directory Bindings.
  directory_bindings = [
    { id = "1" },
  ]

  # Options ▸ EFI Password.
  efi_password = {
    of_mode                = "command"
    of_password            = "EFI-pw-1!"
    of_password_wo_version = 1
  }

  # Options ▸ Restart Options.
  restart_options = {
    no_user_logged_in = "Restart immediately"
    user_logged_in    = "Restart if a package or update requires it"
  }

  # Options ▸ Files and Processes.
  files_and_processes = {
    search_by_path       = "/tmp/sentinel"
    delete_file_if_found = true
    execute_command      = "/usr/local/bin/post-flight.sh"
  }
}
