# Registers Jamf Pro with a Jamf Protect instance (Settings → Jamf apps →
# Jamf Protect). Singleton — one registration per tenant. Creating the
# resource validates the credentials live against Protect and triggers an
# initial plans sync; destroying it unregisters (configuration profiles
# already created from Protect plans remain in Jamf Pro).
#
# `password` is `WriteOnly` — sent to Jamf Pro on writes but never persisted
# in Terraform state. Bump `password_wo_version` to re-register with a rotated
# password on the next apply. Changing `api_url` or `client_id` also
# re-registers in place.
resource "jamfplatform_pro_jamf_protect" "example" {
  api_url             = "https://example.protect.jamfcloud.com/graphql"
  client_id           = "REPLACE-WITH-PROTECT-API-CLIENT-ID"
  password            = sensitive("change-me")
  password_wo_version = 1

  # "Automatically deploy the Jamf Protect PKG with plans" — server default false.
  auto_install = false
}
