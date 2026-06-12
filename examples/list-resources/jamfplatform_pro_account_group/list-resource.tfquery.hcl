# List every Jamf Pro administrator account group.
list "jamfplatform_pro_account_group" "all" {
  provider = jamfplatform
}

# List account groups whose display name contains "Admin" (case-insensitive).
list "jamfplatform_pro_account_group" "admin_like" {
  provider = jamfplatform

  config {
    filter = {
      name_substring = "Admin"
    }
  }
}
