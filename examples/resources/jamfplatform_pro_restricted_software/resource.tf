# Minimal restricted software record. name and process_name are required;
# match-exact defaults to true and the action toggles default to false.
resource "jamfplatform_pro_restricted_software" "minimal" {
  general = {
    name         = "Block iTunes"
    process_name = "iTunes.app"
  }
}

# Kill the process on sight, notify admins, and show a message. Scoped to every
# computer, with a couple of local users excluded.
resource "jamfplatform_pro_restricted_software" "block_chess" {
  general = {
    name                                 = "Block Chess"
    process_name                         = "Chess.app"
    restrict_exact_process_name          = true
    send_email_notification_on_violation = true
    kill_process                         = true
    delete_application                   = false
    display_message                      = "Chess is not permitted on managed devices. Contact the help desk."
  }

  scope = {
    targets = {
      all_computers = true
    }

    exclusions = {
      # Free-text local usernames, not Jamf Pro object IDs.
      directory_service_or_local_user_names = ["labadmin", "kiosk"]
    }
  }
}

# Scoped to a computer group sourced from a Platform Services device group via
# .jamf_pro_id, excluding a department.
resource "jamfplatform_pro_restricted_software" "scoped" {
  general = {
    name         = "Block Solitaire"
    process_name = "Solitaire.app"
  }

  scope = {
    targets = {
      computer_group_ids = [
        jamfplatform_device_group.engineering.jamf_pro_id,
      ]
    }

    exclusions = {
      department_ids = [jamfplatform_pro_department.it.id]
    }
  }
}
