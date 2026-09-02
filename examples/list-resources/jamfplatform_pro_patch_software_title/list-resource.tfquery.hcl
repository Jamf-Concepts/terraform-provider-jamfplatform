# List every Jamf Pro patch software title. NOTE: with include_resource the
# returned objects leave available_versions and extension_attributes null —
# each would cost one extra Jamf Pro call per title; read a single title with
# the data source when you need them.
list "jamfplatform_pro_patch_software_title" "all" {
  provider = jamfplatform
}

# List patch software titles whose display name contains the substring "Adobe"
# (case-insensitive).
list "jamfplatform_pro_patch_software_title" "by_name" {
  provider = jamfplatform

  config {
    filter = {
      name_substring = "Adobe"
    }
  }
}
