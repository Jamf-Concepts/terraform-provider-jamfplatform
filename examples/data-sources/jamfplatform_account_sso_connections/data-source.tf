# Every single sign-on connection in the organization.
data "jamfplatform_account_sso_connections" "all" {}

output "connections_by_provider" {
  value = {
    for c in data.jamfplatform_account_sso_connections.all.sso_connections :
    c.name => c.connection_type
  }
}

# Connections created through the Microsoft admin consent flow cannot be managed
# by Terraform — Jamf owns their client registration — so they are worth knowing
# about before writing configuration.
output "console_managed_connections" {
  value = [
    for c in data.jamfplatform_account_sso_connections.all.sso_connections :
    c.name if c.consent_flow
  ]
}
