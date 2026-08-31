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

# The UUID of an existing Jamf Pro API client permitted to read and update AD CS
# certificate jobs — the privileges Jamf Pro called "Read AD CS Certificate
# Jobs" and "Update AD CS Certificate Jobs" before the Platform API GA, which
# this provider has no Jamf Account permission recorded for. API clients and
# roles are created in Jamf Account, not by this provider.
variable "adcs_api_client_id" {
  type = string
}

# AD CS integration — OUTBOUND mode.
# An AD CS Connector polls Jamf Pro using the API client above.
resource "jamfplatform_pro_pki_adcs" "outbound" {
  connector_mode = "OUTBOUND"
  display_name   = "AD CS — Outbound"
  ca_name        = "Example Issuing CA"
  fqdn           = "adcs.example.com"
  api_client_id  = var.adcs_api_client_id
}
