# List every Jamf Pro iBeacon.
list "jamfplatform_pro_ibeacon" "all" {
  provider = jamfplatform
}

# List iBeacons whose name contains the substring "Reception" (case-insensitive).
list "jamfplatform_pro_ibeacon" "reception_like" {
  provider = jamfplatform

  config {
    filter = {
      name_substring = "Reception"
    }
  }
}
