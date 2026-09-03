# Look a connection up by the identifier Jamf assigned it.
data "jamfplatform_account_sso_connection" "by_id" {
  id = "con_XXXXXXXXXXXXXXXX"
}

# Or by the name Jamf holds for it, which may be a uniquified form of the name it
# was created with — jamfplatform_account_sso_connections reports the stored
# names. Jamf does not require them to be unique, so this reports an error rather
# than choosing when more than one matches.
data "jamfplatform_account_sso_connection" "corp" {
  name = "CorpOIDC"
}

output "domains_signed_in_by_this_connection" {
  value = data.jamfplatform_account_sso_connection.corp.domains
}

# The products this connection is enabled for. Only the product names are
# reported — the tenants within each product are not.
output "products" {
  value = data.jamfplatform_account_sso_connection.corp.enabled_product_names
}

# Connections built with Microsoft's admin-consent flow in the Jamf Account
# console have no client registration of their own and cannot be written back, so
# the jamfplatform_account_sso_connection resource refuses to manage one. Reading
# it here takes no ownership of it, and this is how to tell one apart.
output "console_managed" {
  value = data.jamfplatform_account_sso_connection.corp.consent_flow
}
