# List every Jamf Pro disk encryption configuration.
list "jamfplatform_pro_disk_encryption_configuration" "all" {
  provider = jamfplatform
}

# List disk encryption configurations whose name contains the substring
# "institutional" (case-insensitive).
list "jamfplatform_pro_disk_encryption_configuration" "institutional_configs" {
  provider = jamfplatform

  config {
    filter = {
      name_substring = "institutional"
    }
  }
}
