# Copyright Jamf Software LLC 2026
# SPDX-License-Identifier: MPL-2.0

# Preview feature: Jamf Pro's Venafi integration is a preview and may change in a future release.

# Minimal: name only. Jamf Pro mints a public key (jamf_public_key) on create.
resource "jamfplatform_pro_pki_venafi" "minimal" {
  name = "Venafi CA"
}

# Full: proxy server, OAuth client, revocation, and a rotatable refresh token.
resource "jamfplatform_pro_pki_venafi" "example" {
  name = "Corporate Venafi CA"

  # host:port with NO scheme (a https:// prefix is rejected by Jamf Pro).
  proxy_address      = "venafi-proxy.example.com:8443"
  client_id          = "REPLACE_WITH_VENAFI_CLIENT_ID"
  revocation_enabled = true

  # The Venafi refresh token is write-only — sent to Jamf Pro but never stored
  # in Terraform state. Bump refresh_token_wo_version to re-send / rotate it.
  refresh_token_wo         = "REPLACE_WITH_VENAFI_REFRESH_TOKEN"
  refresh_token_wo_version = 1

  # The PKI Proxy Server's PUBLIC certificate chain (round-trips byte-exact).
  # Set to "" to remove it.
  proxy_trust_store = file("${path.module}/proxy-public.pem")

  # Bump to regenerate the Jamf-minted public key (jamf_public_key).
  jamf_public_key_rotation = 1
}
