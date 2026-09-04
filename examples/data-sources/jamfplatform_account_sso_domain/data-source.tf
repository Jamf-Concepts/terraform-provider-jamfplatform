# Look a claimed domain up by name to read its verification state, and to see
# which single sign-on connections currently use it.
data "jamfplatform_account_sso_domain" "corp" {
  domain = "example.com"
}

output "verified" {
  value = data.jamfplatform_account_sso_domain.corp.verification_status
}

# Removing a domain also detaches it from every connection listed here, so this
# is worth reading before destroying a claim.
output "connections_using_this_domain" {
  value = [for c in data.jamfplatform_account_sso_domain.corp.assigned_connections : c.connection_id]
}

# Whether users on this domain can still sign in with a Jamf ID alongside the
# identity provider.
output "jamf_id_sign_in_allowed" {
  value = data.jamfplatform_account_sso_domain.corp.jamf_id_enabled
}
