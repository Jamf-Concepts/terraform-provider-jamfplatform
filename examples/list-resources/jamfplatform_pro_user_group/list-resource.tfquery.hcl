# List every Jamf Pro user group.
list "jamfplatform_pro_user_group" "all" {
  provider = jamfplatform
}

# List user groups whose name contains the substring "Apple" (case-insensitive).
list "jamfplatform_pro_user_group" "apple_like" {
  provider = jamfplatform

  config {
    filter = {
      name_substring = "Apple"
    }
  }
}
