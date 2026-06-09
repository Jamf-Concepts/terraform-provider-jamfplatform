# Jamf Pro DigiCert Trust Lifecycle Manager integration
# (Settings > Global > PKI certificates > Certificate Authorities).
#
# DigiCert TLM is an external certificate authority Jamf Pro uses to issue
# certificates referenced by configuration profiles.
#
# The client certificate (a .p12/.pfx keystore) is supplied through the
# client_certificate block. data_wo and password_wo are WriteOnly: sent to
# Jamf Pro on writes but never stored in Terraform state, and never returned
# on read. DigiCert treats the certificate as all-or-nothing — the provider
# re-sends it only when wo_version changes. Bump wo_version to rotate the
# stored certificate.
resource "jamfplatform_pro_pki_digicert" "example" {
  display_name       = "DigiCert TLM"
  host_name          = "one.digicert.com"
  revocation_enabled = false

  client_certificate = {
    data_wo     = filebase64("${path.module}/digicert-client.p12")
    password_wo = sensitive(var.digicert_keystore_password)
    filename    = "digicert-client.p12"
    wo_version  = 1
  }
}

# Read-only certificate metadata returned by Jamf Pro:
#   jamfplatform_pro_pki_digicert.example.client_certificate_details.serial_number
#   jamfplatform_pro_pki_digicert.example.client_certificate_details.subject
#   jamfplatform_pro_pki_digicert.example.client_certificate_details.issuer
#   jamfplatform_pro_pki_digicert.example.client_certificate_details.expiration_date
