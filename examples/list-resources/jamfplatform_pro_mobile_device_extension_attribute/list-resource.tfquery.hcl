# List every Jamf Pro mobile device extension attribute.
list "jamfplatform_pro_mobile_device_extension_attribute" "all" {
  provider = jamfplatform
}

# List mobile device extension attributes whose name contains "Role"
# (case-insensitive).
list "jamfplatform_pro_mobile_device_extension_attribute" "role_like" {
  provider = jamfplatform

  config {
    filter = {
      name_substring = "Role"
    }
  }
}
