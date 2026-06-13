# App Request form fields (Settings → Self Service → App Request → App Request Form).
#
# Each field is an independent record. Fields are shown to the requester in
# ascending `priority` order.

resource "jamfplatform_pro_app_request_form_field" "reason" {
  title       = "Reason for request"
  description = "Tell us why you need this app."
  priority    = 1
}

resource "jamfplatform_pro_app_request_form_field" "cost_center" {
  title    = "Cost center"
  priority = 2
}
