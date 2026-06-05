# List every Jamf Pro mobile device provisioning profile.
list "jamfplatform_pro_mobile_device_provisioning_profile" "all" {
  provider = jamfplatform
}

# List profiles whose name contains the substring "In-House" (case-insensitive).
list "jamfplatform_pro_mobile_device_provisioning_profile" "in_house_like" {
  provider = jamfplatform

  config {
    filter = {
      name_substring = "In-House"
    }
  }
}
