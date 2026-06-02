# List every Jamf Pro advanced volume purchasing content search.
list "jamfplatform_pro_advanced_volume_purchasing_content_search" "all" {
  provider = jamfplatform
}

# List advanced volume purchasing content searches whose name contains the
# substring "Office" (case-insensitive).
list "jamfplatform_pro_advanced_volume_purchasing_content_search" "office_like" {
  provider = jamfplatform

  config {
    filter = {
      name_substring = "Office"
    }
  }
}
