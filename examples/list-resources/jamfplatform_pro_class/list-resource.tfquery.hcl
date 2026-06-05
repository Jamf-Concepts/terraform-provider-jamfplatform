# List every Jamf Pro class.
list "jamfplatform_pro_class" "all" {
  provider = jamfplatform
}

# List classes whose name contains the substring "Biology" (case-insensitive).
list "jamfplatform_pro_class" "biology_like" {
  provider = jamfplatform

  config {
    filter = {
      name_substring = "Biology"
    }
  }
}
