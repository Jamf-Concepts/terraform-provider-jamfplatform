# Copyright Jamf Software LLC 2026
# SPDX-License-Identifier: MPL-2.0

# AD CS (Active Directory Certificate Services) integration — INBOUND mode.
# Jamf Pro reaches an AD CS Connector at adcs_url, presenting a client
# certificate (.pfx/.p12) and trusting a server certificate (.pem/.cer).
#
# Certificate bytes and the client certificate password are WriteOnly: supply
# them with filebase64(...) and bump the block's wo_version to rotate the stored
# certificate. They are never persisted in Terraform state.
resource "jamfplatform_pro_pki_adcs" "inbound" {
  connector_mode     = "INBOUND"
  display_name       = "AD CS — Inbound"
  ca_name            = "Example Issuing CA"
  fqdn               = "adcs.example.com"
  adcs_url           = "connector.example.com" # no scheme
  revocation_enabled = true

  server_certificate = {
    data_wo    = filebase64("${path.module}/certs/adcs-server.pem")
    filename   = "adcs-server.pem"
    wo_version = 1
  }

  client_certificate = {
    data_wo     = filebase64("${path.module}/certs/adcs-client.p12")
    password_wo = var.adcs_client_p12_password
    filename    = "adcs-client.p12"
    wo_version  = 1
  }
}

variable "adcs_client_p12_password" {
  type      = string
  sensitive = true
}

# AD CS integration — OUTBOUND mode.
# An AD CS Connector polls Jamf Pro using a Jamf Pro API client that holds the
# "Read AD CS Certificate Jobs" and "Update AD CS Certificate Jobs" privileges.
resource "jamfplatform_pro_api_role" "adcs" {
  display_name = "AD CS Connector"
  privileges = [
    "Read AD CS Certificate Jobs",
    "Update AD CS Certificate Jobs",
  ]
}

resource "jamfplatform_pro_api_client" "adcs" {
  display_name = "AD CS Connector"
  api_roles    = [jamfplatform_pro_api_role.adcs.display_name]
  enabled      = true
}

resource "jamfplatform_pro_pki_adcs" "outbound" {
  connector_mode = "OUTBOUND"
  display_name   = "AD CS — Outbound"
  ca_name        = "Example Issuing CA"
  fqdn           = "adcs.example.com"
  api_client_id  = jamfplatform_pro_api_client.adcs.client_id
}
