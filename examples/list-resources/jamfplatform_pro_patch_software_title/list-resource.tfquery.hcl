# List every Jamf Pro patch software title.
list "jamfplatform_pro_patch_software_title" "all" {
  provider = jamfplatform
}

# List patch software titles whose name_id (catalog key) contains the substring
# "285" (case-insensitive). NOTE: the classic list response surfaces no display
# name through the SDK, so the filter matches name_id, not the display name.
list "jamfplatform_pro_patch_software_title" "by_name_id" {
  provider = jamfplatform

  config {
    filter = {
      name_substring = "285"
    }
  }
}
