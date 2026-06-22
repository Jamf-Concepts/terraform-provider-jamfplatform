action "jamfplatform_pro_send_blank_push" "nudge" {
  config {
    # Source management IDs from a device group's members, or list serials.
    management_ids = [
      "00000000-0000-0000-0000-000000000000",
    ]
    serial_numbers = [
      "C02XXXXXXXXX",
    ]
  }
}
