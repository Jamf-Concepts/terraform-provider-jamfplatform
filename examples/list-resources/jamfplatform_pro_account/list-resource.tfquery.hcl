# List every Jamf Pro administrator account.
list "jamfplatform_pro_account" "all" {
  provider = jamfplatform
}

# List accounts whose username contains "admin" (case-insensitive).
list "jamfplatform_pro_account" "admin_like" {
  provider = jamfplatform

  config {
    filter = {
      name_substring = "admin"
    }
  }
}
