# List every Jamf Pro patch external source.
list "jamfplatform_pro_patch_external_source" "all" {
  provider = jamfplatform
}

# List patch external sources whose name contains the substring "Auto"
# (case-insensitive).
list "jamfplatform_pro_patch_external_source" "auto_like" {
  provider = jamfplatform

  config {
    filter = {
      name_substring = "Auto"
    }
  }
}
