# List every Jamf Pro allowed file extension.
list "jamfplatform_pro_allowed_file_extension" "all" {
  provider = jamfplatform
}

# List allowed file extensions whose value contains the substring "doc"
# (case-insensitive).
list "jamfplatform_pro_allowed_file_extension" "doc_like" {
  provider = jamfplatform

  config {
    filter = {
      name_substring = "doc"
    }
  }
}
