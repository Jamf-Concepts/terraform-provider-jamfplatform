# List every Jamf Pro mobile device enrollment profile.
list "jamfplatform_pro_mobile_device_enrollment_profile" "all" {
  provider = jamfplatform
}

# Filter by case-insensitive name substring.
list "jamfplatform_pro_mobile_device_enrollment_profile" "configurator_like" {
  provider = jamfplatform

  config {
    filter = {
      name_substring = "Configurator"
    }
  }
}
