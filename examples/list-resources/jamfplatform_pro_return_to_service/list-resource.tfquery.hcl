# List every Jamf Pro Return to Service configuration.
list "jamfplatform_pro_return_to_service" "all" {
  provider = jamfplatform
}

# List Return to Service configurations whose display name contains the
# substring "Front Desk" (case-insensitive).
list "jamfplatform_pro_return_to_service" "front_desk_like" {
  provider = jamfplatform

  config {
    filter = {
      name_substring = "Front Desk"
    }
  }
}
