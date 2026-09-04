# App Request settings (Settings → Self Service → App Request).
#
# One record per tenant. This resource adopts the existing settings on first
# apply. Enabling App Requests requires a static user group (the requesters), at
# least one App Request form field, and an SMTP server for the approval emails.

resource "jamfplatform_pro_user_group" "app_request_requesters" {
  name       = "App Request Requesters"
  group_type = "static"
}

resource "jamfplatform_pro_app_request_form_field" "reason" {
  title       = "Reason for request"
  description = "Why do you need this app?"
  priority    = 1
}

resource "jamfplatform_pro_app_request_settings" "example" {
  enabled                 = true
  app_store_locale        = "US" # or the literal "deviceLocale" to follow each device
  approver_emails         = ["it-approvals@example.com"]
  requester_user_group_id = tonumber(jamfplatform_pro_user_group.app_request_requesters.id)

  # The form field must exist before App Requests can be enabled.
  depends_on = [jamfplatform_pro_app_request_form_field.reason]
}
