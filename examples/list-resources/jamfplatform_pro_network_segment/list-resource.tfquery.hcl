# List every Jamf Pro network segment.
list "jamfplatform_pro_network_segment" "all" {
  provider = jamfplatform
}

# List network segments whose name contains the substring "HQ" (case-insensitive).
list "jamfplatform_pro_network_segment" "hq_like" {
  provider = jamfplatform

  config {
    filter = {
      name_substring = "HQ"
    }
  }
}
