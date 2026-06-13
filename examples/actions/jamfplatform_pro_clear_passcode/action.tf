action "jamfplatform_pro_clear_passcode" "clear" {
  config {
    serial_number = "DMPXXXXXXXXX"

    # unlock_token is looked up automatically. Supply it only to override
    # the lookup (required for some unsupervised devices).
    # unlock_token = "..."
  }
}
