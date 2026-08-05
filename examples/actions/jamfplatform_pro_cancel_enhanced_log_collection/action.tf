# Stops an in-progress AppleCare enhanced log collection session. The outcome for
# each device is reported in that device's management history.
#
# Requires Jamf Pro 11.30 or later, and a device running iOS, iPadOS, tvOS or
# macOS 27.0 or later.
action "jamfplatform_pro_cancel_enhanced_log_collection" "stop_collection" {
  config {
    # Set management_ids (the `id` values from the jamfplatform_devices/
    # jamfplatform_device data sources) and/or serial_numbers. The two are
    # additive, and every listed device is commanded in a single request.
    serial_numbers = ["C02XXXXXXXXX"]
  }
}

# Unlike the trigger action this command carries no payload, so there is no
# per-device value to misapply and batching is unconditionally safe.
action "jamfplatform_pro_cancel_enhanced_log_collection" "stop_all" {
  config {
    management_ids = [
      "11111111-1111-1111-1111-111111111111",
      "22222222-2222-2222-2222-222222222222",
    ]
  }
}
