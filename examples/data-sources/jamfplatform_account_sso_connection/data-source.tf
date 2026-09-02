# Look a connection up by the identifier Jamf assigned it.
data "jamfplatform_account_sso_connection" "by_id" {
  id = "con_XXXXXXXXXXXXXXXX"
}

# Or by name. Jamf does not require connection names to be unique, so this
# reports an error rather than choosing when more than one matches.
data "jamfplatform_account_sso_connection" "corp" {
  name = "Corp OIDC"
}

output "domains_signed_in_by_this_connection" {
  value = data.jamfplatform_account_sso_connection.corp.domains
}

# The products this connection is enabled for. Only the product names are
# reported — the tenants within each product are not.
output "products" {
  value = data.jamfplatform_account_sso_connection.corp.enabled_product_names
}
