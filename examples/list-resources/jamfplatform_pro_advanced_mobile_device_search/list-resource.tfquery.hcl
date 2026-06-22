# List every Jamf Pro advanced mobile device search.
list "jamfplatform_pro_advanced_mobile_device_search" "all" {
  provider = jamfplatform
}

# List advanced mobile device searches whose name contains the substring
# "Unmanaged" (case-insensitive).
list "jamfplatform_pro_advanced_mobile_device_search" "unmanaged_like" {
  provider = jamfplatform

  config {
    filter = {
      name_substring = "Unmanaged"
    }
  }
}
