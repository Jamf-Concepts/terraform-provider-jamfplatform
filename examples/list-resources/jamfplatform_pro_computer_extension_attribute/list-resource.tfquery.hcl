# List every Jamf Pro computer extension attribute.
list "jamfplatform_pro_computer_extension_attribute" "all" {
  provider = jamfplatform
}

# List computer extension attributes whose name contains "Asset"
# (case-insensitive).
list "jamfplatform_pro_computer_extension_attribute" "asset_like" {
  provider = jamfplatform

  config {
    filter = {
      name_substring = "Asset"
    }
  }
}
