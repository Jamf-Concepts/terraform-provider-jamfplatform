# Jamf Pro GSX Connection settings (Settings > Global > GSX connection).
#
# Connects Jamf Pro to Apple's Global Service Exchange (GSX) for warranty,
# repair, and purchase-date lookups. Singleton — one record per tenant.
#
# Requires a valid Apple-registered GSX certificate. Every apply re-validates
# the certificate, token, and account against Apple's live GSX service.
#
# The three secrets are `Required` + `WriteOnly`: sent to Jamf Pro on every
# apply but never stored in Terraform state, and never returned on read. The
# GSX API mandates them on every write, so they must always be present in
# config — to rotate, change the value here.
resource "jamfplatform_pro_gsx_connection_settings" "this" {
  enabled                = true
  username               = "gsx-admin@example.com"
  service_account_number = "1234567890"
  ship_to_number         = "54321"

  token_wo             = sensitive(var.gsx_api_token)
  keystore_bytes_wo    = filebase64("${path.module}/gsx-certificate.p12")
  keystore_password_wo = sensitive(var.gsx_keystore_password)
  keystore_name        = "gsx-certificate.p12"
}
