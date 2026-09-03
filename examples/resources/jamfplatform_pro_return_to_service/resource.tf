# Return to Service configuration. Pairs a display name with a Wi-Fi
# configuration profile. When a device is erased with Return to Service, the
# referenced Wi-Fi profile lets it automatically rejoin a network and re-enrol
# into Jamf Pro without manual setup.

# A mobile device configuration profile carrying a Wi-Fi payload. Its ID is what
# the Return to Service configuration references.
resource "jamfplatform_pro_mobile_device_configuration_profile" "front_desk_wifi" {
  general = {
    name     = "Front Desk Wi-Fi"
    payloads = file("${path.module}/front-desk-wifi.mobileconfig")
  }
}

resource "jamfplatform_pro_return_to_service" "front_desk" {
  display_name    = "Front Desk iPads"
  wifi_profile_id = jamfplatform_pro_mobile_device_configuration_profile.front_desk_wifi.id
}
