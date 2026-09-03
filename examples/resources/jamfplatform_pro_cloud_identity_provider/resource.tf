# Google Secure LDAP Cloud Identity Provider.
#
# The keystore `file` (base64 of the PKCS#12 client certificate) and
# `password` are `WriteOnly`, sent to Jamf Pro on writes but never persisted
# in Terraform state. Bump `keystore.wo_version` to re-upload (rotate) the
# certificate on a later apply. Omit `mappings` to let Jamf Pro generate the
# standard Google defaults.
#
# Attribute names mirror the labels used in the Jamf Pro admin UI.
resource "jamfplatform_pro_cloud_identity_provider" "google" {
  display_name  = "Google Workspace"
  provider_name = "GOOGLE"

  google = {
    server = {
      domain_name = "example.com"
      # server_url, port, connection_type, timeouts, use_wildcards and
      # enabled all default to Jamf Pro's standard Google Secure LDAP values
      # when omitted.

      keystore = {
        file       = filebase64("${path.module}/google-ldap.p12")
        password   = sensitive(var.google_ldap_keystore_password)
        wo_version = 1
      }
    }
  }
}

# Microsoft Entra ID (Azure AD) Cloud Identity Provider.
#
# After the first apply you must complete the manual "refresh consent" step in
# the Jamf Pro admin UI (sign into Entra ID and authorise the Jamf cloud
# connector). Until consent exists the connection is inactive and later updates
# are rejected by Entra.
resource "jamfplatform_pro_cloud_identity_provider" "entra" {
  display_name  = "Entra ID"
  provider_name = "ENTRA_ID"

  entra_id = {
    tenant_id = "d5749c84-5cc5-4691-a187-4545c02ff915" # your Entra ID tenant GUID
    # search_timeout, enabled, transitivity flags and the membership user
    # field all carry sensible defaults when omitted. Omit `mappings` to let
    # Jamf Pro generate the Entra ID defaults.
  }
}

variable "google_ldap_keystore_password" {
  type        = string
  sensitive   = true
  description = "Password protecting the Google Secure LDAP PKCS#12 keystore."
}
