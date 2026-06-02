# List every Jamf Pro advanced computer search.
list "jamfplatform_pro_advanced_computer_search" "all" {
  provider = jamfplatform
}

# List advanced computer searches whose name contains the substring "Lab"
# (case-insensitive).
list "jamfplatform_pro_advanced_computer_search" "lab_like" {
  provider = jamfplatform

  config {
    filter = {
      name_substring = "Lab"
    }
  }
}
