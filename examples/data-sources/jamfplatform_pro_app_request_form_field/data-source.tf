# Look up an App Request form field by ID or by title (exactly one).
# Titles are not unique — a by-title lookup that matches more than one field errors.

data "jamfplatform_pro_app_request_form_field" "by_id" {
  id = "12"
}

data "jamfplatform_pro_app_request_form_field" "by_title" {
  title = "Reason for request"
}

output "app_request_form_field_by_id" {
  value = data.jamfplatform_pro_app_request_form_field.by_id
}

output "app_request_form_field_by_title" {
  value = data.jamfplatform_pro_app_request_form_field.by_title
}
