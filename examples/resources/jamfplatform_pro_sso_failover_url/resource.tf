# SSO failover URL. Treat the resulting `failover_url` as a credential.
# Rotate by bumping `regeneration_trigger`.
resource "jamfplatform_pro_sso_failover_url" "this" {
  regeneration_trigger = "v1"
}

output "sso_failover_url" {
  value     = jamfplatform_pro_sso_failover_url.this.failover_url
  sensitive = true
}
