resource "jamfplatform_pro_mobile_device_enrollment_profile" "configurator" {
  name        = "Configurator Enrollment Profile"
  description = "OTA / Apple Configurator enrollment profile"

  # Optional: scope enrolled devices to a site.
  site_id = jamfplatform_pro_site.example.id

  location = {
    username  = "labuser"
    real_name = "Lab User"
    building  = "HQ"
    room      = "101"
  }

  purchasing = {
    is_purchased    = true
    vendor          = "Apple"
    applecare_id    = "AC-123456"
    life_expectancy = 3
  }
}

# The invitation code and UUID are minted by Jamf Pro.
output "enrollment_invitation" {
  value = jamfplatform_pro_mobile_device_enrollment_profile.configurator.invitation
}

# Attachments are read-only — manage them in the Jamf Pro admin UI.
output "enrollment_attachments" {
  value = jamfplatform_pro_mobile_device_enrollment_profile.configurator.attachments
}
