action "jamfplatform_pro_flush_policy_logs" "flush_old" {
  config {
    policy_id = "123"
    interval  = "Six+Months"
  }
}
