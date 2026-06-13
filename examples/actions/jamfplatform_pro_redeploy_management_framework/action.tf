action "jamfplatform_pro_redeploy_management_framework" "redeploy" {
  config {
    # Provide a management_id (the `id` from the jamfplatform_devices/
    # jamfplatform_device data sources) or a serial_number.
    serial_number = "C02XXXXXXXXX"
  }
}
