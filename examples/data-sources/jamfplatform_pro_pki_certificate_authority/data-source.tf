# Read the active Certificate Authority (the Jamf Pro built-in CA on most tenants).
data "jamfplatform_pro_pki_certificate_authority" "active" {}

output "active_ca_pem" {
  value = data.jamfplatform_pro_pki_certificate_authority.active.pem
}

output "active_ca_expiry_epoch" {
  value = data.jamfplatform_pro_pki_certificate_authority.active.not_after
}

# Read a specific Certificate Authority by id.
data "jamfplatform_pro_pki_certificate_authority" "by_id" {
  id = "1"
}
