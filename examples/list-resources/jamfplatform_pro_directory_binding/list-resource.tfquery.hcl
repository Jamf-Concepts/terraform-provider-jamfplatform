# List every Jamf Pro directory binding.
list "jamfplatform_pro_directory_binding" "all" {
  provider = jamfplatform
}

# List directory bindings whose name contains the substring "prod"
# (case-insensitive).
list "jamfplatform_pro_directory_binding" "prod_bindings" {
  provider = jamfplatform

  config {
    filter = {
      name_substring = "prod"
    }
  }
}
