# Flush a policy's logs older than six months.
action "jamfplatform_pro_flush_policy_logs" "flush_old" {
  config {
    policy_id = "123"
    quantity  = "Six"
    period    = "Months"
  }
}

# quantity = "Zero" flushes every log for the policy, because no log is younger
# than zero days. There is no "Four" or "Five" quantity.
action "jamfplatform_pro_flush_policy_logs" "flush_all" {
  config {
    policy_id = "123"
    quantity  = "Zero"
    period    = "Days"
  }
}

resource "terraform_data" "log_maintenance" {
  lifecycle {
    action_trigger {
      events  = [after_create]
      actions = [action.jamfplatform_pro_flush_policy_logs.flush_old]
    }
  }
}
