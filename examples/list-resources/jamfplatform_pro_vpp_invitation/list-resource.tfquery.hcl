# List every Jamf Pro VPP invitation.
list "jamfplatform_pro_vpp_invitation" "all" {
  provider = jamfplatform
}

# Filter by case-insensitive name substring.
list "jamfplatform_pro_vpp_invitation" "self_service_like" {
  provider = jamfplatform

  config {
    filter = {
      name_substring = "Self Service"
    }
  }
}
