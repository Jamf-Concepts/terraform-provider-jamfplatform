action "jamfplatform_pro_device_lock" "lock_lost_laptops" {
  config {
    # Set management_ids (the `id` values from the jamfplatform_devices/
    # jamfplatform_device data sources) and/or serial_numbers. The two are
    # additive, and every listed device is commanded in a single request.
    serial_numbers = [
      "C02XXXXXXXXX",
      "C02YYYYYYYYY",
    ]
    management_ids = [
      "00000000-0000-0000-0000-000000000000",
    ]

    # These apply to every device in the batch.
    message      = "This device has been locked. Please contact IT."
    phone_number = "+1-555-0100"
    pin          = "123456" # exactly six characters; computers only
  }
}

# Management IDs are the cheaper selector: serial numbers must first be looked up
# to find the matching device.
action "jamfplatform_pro_device_lock" "lock_by_id" {
  config {
    management_ids = [
      "11111111-1111-1111-1111-111111111111",
      "22222222-2222-2222-2222-222222222222",
    ]
    pin = "654321"
  }
}
