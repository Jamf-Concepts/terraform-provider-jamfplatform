# List every Jamf Pro removable MAC address.
list "jamfplatform_pro_removable_mac_address" "all" {
  provider = jamfplatform
}

# List removable MAC addresses whose value contains the substring "C9"
# (case-insensitive).
list "jamfplatform_pro_removable_mac_address" "c9_like" {
  provider = jamfplatform

  config {
    filter = {
      name_substring = "C9"
    }
  }
}
