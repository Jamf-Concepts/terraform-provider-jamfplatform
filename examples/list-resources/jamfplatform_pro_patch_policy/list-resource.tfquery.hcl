# List every Jamf Pro patch policy.
list "jamfplatform_pro_patch_policy" "all" {
  provider = jamfplatform
}

# List patch policies whose display name contains the substring "8x8"
# (case-insensitive). Unlike patch software titles, the classic patch policies
# list response carries a display name, so the filter matches the policy name.
list "jamfplatform_pro_patch_policy" "by_name" {
  provider = jamfplatform

  config {
    filter = {
      name_substring = "8x8"
    }
  }
}
