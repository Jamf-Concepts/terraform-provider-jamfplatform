# App Request settings (Settings → Self Service → App Request).
#
# Singleton — one record per tenant. This resource adopts the existing
# settings on first apply. Enabling App Requests requires a static user group
# (the requesters) and a configured SMTP server (so approval emails can be sent).

resource "jamfplatform_pro_user_group" "app_request_requesters" {
  name       = "App Request Requesters"
  group_type = "static"
}

resource "jamfplatform_pro_app_request_settings" "example" {
  enabled                 = true
  app_store_locale        = "US" # or the literal "deviceLocale" to follow each device
  approver_emails         = ["it-approvals@example.com"]
  requester_user_group_id = tonumber(jamfplatform_pro_user_group.app_request_requesters.id)
}
