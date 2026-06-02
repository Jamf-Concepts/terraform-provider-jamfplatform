# List every Jamf Pro advanced user search.
list "jamfplatform_pro_advanced_user_search" "all" {
  provider = jamfplatform
}

# List advanced user searches whose name contains the substring "email"
# (case-insensitive).
list "jamfplatform_pro_advanced_user_search" "email_like" {
  provider = jamfplatform

  config {
    filter = {
      name_substring = "email"
    }
  }
}
