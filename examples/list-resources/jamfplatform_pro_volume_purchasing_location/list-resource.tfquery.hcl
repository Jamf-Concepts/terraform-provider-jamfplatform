# List every Jamf Pro Volume Purchasing (VPP) location.
list "jamfplatform_pro_volume_purchasing_location" "all" {
  provider = jamfplatform
}

# List VPP locations whose name contains the substring "prod"
# (case-insensitive). The list endpoint returns the slim ListView shape per
# row; setting include_resource = true triggers a follow-up GET per row to
# populate the `content` catalog. Identity-only listing stays a single round
# trip.
list "jamfplatform_pro_volume_purchasing_location" "prod_locations" {
  provider = jamfplatform

  config {
    filter = {
      name_substring = "prod"
    }
  }
}
