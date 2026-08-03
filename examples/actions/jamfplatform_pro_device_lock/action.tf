action "jamfplatform_pro_device_lock" "lock_lost_laptop" {
  config {
    # Set exactly one of management_id (the `id` from the jamfplatform_devices/
    # jamfplatform_device data sources) or serial_number.
    serial_number = "C02XXXXXXXXX"

    message      = "This device has been locked. Please contact IT."
    phone_number = "+1-555-0100"
    pin          = "123456" # exactly six characters; computers only
  }
}
