# List every Jamf Pro App Request form field.
list "jamfplatform_pro_app_request_form_field" "all" {
  provider = jamfplatform
}

# List App Request form fields whose title contains the substring "reason"
# (case-insensitive). Filtering is applied client-side after the full list is fetched.
list "jamfplatform_pro_app_request_form_field" "reason_like" {
  provider = jamfplatform

  config {
    filter = {
      name_substring = "reason"
    }
  }
}
