# List every Jamf Pro site.
list "jamfplatform_pro_site" "all" {
  provider = jamfplatform
}

# List sites whose name contains the substring "Primary" (case-insensitive).
list "jamfplatform_pro_site" "primary_like" {
  provider = jamfplatform

  config {
    filter = {
      name_substring = "Primary"
    }
  }
}
