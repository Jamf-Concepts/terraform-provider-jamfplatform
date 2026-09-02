# Claiming a domain is step one of two. Jamf will not treat the domain as yours
# until a TXT record proving ownership is published in its DNS, and only a
# verified domain can be attached to a single sign-on connection.
resource "jamfplatform_account_sso_domain" "corp" {
  domain = "example.com"
}

# The record Jamf looks for. Publishing it is the one step that happens outside
# Jamf — verification_txt_record is the complete value to publish, so there is no
# string to assemble by hand and get subtly wrong. Use @ as the host and a TTL of
# 86400. Where your DNS is managed by Terraform too, feed this straight into the
# relevant record resource and a single apply does both halves.
output "dns_record_to_publish" {
  value = jamfplatform_account_sso_domain.corp.verification_txt_record
}

# Verification is a separate step, because it depends on DNS having propagated —
# which can take hours and is outside Jamf's control. Trigger it with the
# jamfplatform_account_sso_domain_verify action once the record is live.
output "verification_state" {
  value = jamfplatform_account_sso_domain.corp.verification_status
}

# A claim lapses fourteen days after it was last verified, and Jamf re-checks in
# the background, so leave the TXT record in place permanently rather than
# removing it once the domain first verifies.
output "verification_expires" {
  value = jamfplatform_account_sso_domain.corp.verification_expires_at
}
