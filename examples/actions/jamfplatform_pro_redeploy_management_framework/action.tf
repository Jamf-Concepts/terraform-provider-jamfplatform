action "jamfplatform_pro_redeploy_management_framework" "redeploy" {
  config {
    # Provide exactly one of: serial_number, management_id (the `id` from the
    # jamfplatform_devices/jamfplatform_device data sources), or udid.
    serial_number = "C02XXXXXXXXX"
  }
}
