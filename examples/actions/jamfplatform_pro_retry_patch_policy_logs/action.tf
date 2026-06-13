action "jamfplatform_pro_retry_patch_policy_logs" "retry" {
  config {
    patch_policy_id = "123"

    # Omit device_ids to retry all failed devices.
    device_ids = ["456", "789"]
  }
}
