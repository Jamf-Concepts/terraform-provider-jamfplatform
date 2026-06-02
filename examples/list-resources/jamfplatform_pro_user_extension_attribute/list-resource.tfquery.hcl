# List every Jamf Pro user extension attribute.
list "jamfplatform_pro_user_extension_attribute" "all" {
  provider = jamfplatform
}

# List user extension attributes whose name contains "Center"
# (case-insensitive).
list "jamfplatform_pro_user_extension_attribute" "center_like" {
  provider = jamfplatform

  config {
    filter = {
      name_substring = "Center"
    }
  }
}
