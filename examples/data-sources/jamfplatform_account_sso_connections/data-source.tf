# Every single sign-on connection in the organization.
data "jamfplatform_account_sso_connections" "all" {}

# Keyed on the identifier rather than the name: Jamf does not require connection
# names to be unique, so two entries can carry the same one.
output "connection_types_by_id" {
  value = {
    for c in data.jamfplatform_account_sso_connections.all.sso_connections :
    c.id => c.connection_type
  }
}

# Which connection signs people in for which domains, and the products each one
# reaches. Only the product names are reported here — the tenants within each
# product are never returned, and the provider-specific settings are reported one
# connection at a time by the singular data source.
output "connection_coverage" {
  value = [
    for c in data.jamfplatform_account_sso_connections.all.sso_connections : {
      id       = c.id
      name     = c.name
      domains  = c.domains
      products = c.enabled_product_names
    }
  ]
}
