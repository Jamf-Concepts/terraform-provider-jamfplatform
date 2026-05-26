data "jamfplatform_pro_sso_failover_url" "current" {}

output "sso_failover_url" {
  value     = data.jamfplatform_pro_sso_failover_url.current.failover_url
  sensitive = true
}

output "sso_failover_generated_at" {
  value = data.jamfplatform_pro_sso_failover_url.current.generation_time_utc
}
