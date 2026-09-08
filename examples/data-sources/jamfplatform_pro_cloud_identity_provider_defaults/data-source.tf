# The attribute mappings and connection settings Jamf Pro pre-fills when an
# administrator adds a cloud identity provider. Takes no arguments, and reports
# both products whatever the tenant has configured.
data "jamfplatform_pro_cloud_identity_provider_defaults" "jamf" {}

# Seeding an Entra ID connection. A declared mappings block owns all eleven
# fields, so starting from the defaults beats transcribing them.
resource "jamfplatform_pro_cloud_identity_provider" "entra" {
  display_name  = "Entra ID"
  provider_name = "ENTRA_ID"

  entra_id = {
    tenant_id = "d5749c84-5cc5-4691-a187-4545c02ff915" # your Entra ID tenant GUID
    mappings  = data.jamfplatform_pro_cloud_identity_provider_defaults.jamf.entra_id.mappings
  }
}

# Google Secure LDAP takes the defaults one field at a time, because two of them
# cannot be copied: the domain is yours to supply, and additional_search_base has
# to be a distinguished name, which the empty default is not.
resource "jamfplatform_pro_cloud_identity_provider" "google" {
  display_name  = "Google Workspace"
  provider_name = "GOOGLE"

  google = {
    server = {
      domain_name        = "example.com"
      server_url         = data.jamfplatform_pro_cloud_identity_provider_defaults.jamf.google.server.server_url
      port               = data.jamfplatform_pro_cloud_identity_provider_defaults.jamf.google.server.port
      connection_timeout = data.jamfplatform_pro_cloud_identity_provider_defaults.jamf.google.server.connection_timeout
      search_timeout     = data.jamfplatform_pro_cloud_identity_provider_defaults.jamf.google.server.search_timeout

      keystore = {
        file       = filebase64("${path.module}/google-ldap.p12")
        password   = sensitive(var.google_ldap_keystore_password)
        wo_version = 1
      }
    }
  }
}

# What Jamf Pro would map a username to on each side.
output "entra_default_username_mapping" {
  value = data.jamfplatform_pro_cloud_identity_provider_defaults.jamf.entra_id.mappings.user_name
}

output "google_default_username_mapping" {
  value = data.jamfplatform_pro_cloud_identity_provider_defaults.jamf.google.mappings.user_mappings.username
}

variable "google_ldap_keystore_password" {
  type        = string
  sensitive   = true
  description = "Password protecting the Google Secure LDAP PKCS#12 keystore."
}
