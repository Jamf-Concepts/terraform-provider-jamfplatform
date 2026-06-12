# Minimal policy: just an enabled flag and a name. Every other section
# is optional.
resource "jamfplatform_pro_policy" "minimal" {
  general = {
    name    = "tf-acc-minimal-policy"
    enabled = true
  }
}

# Fully-scoped policy demonstrating per-target IDs sourced from sibling
# resources. Interpolate `.jamf_pro_id` on Platform Services device groups —
# `scope.computer_group_ids` takes Jamf Pro IDs as strings.
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
