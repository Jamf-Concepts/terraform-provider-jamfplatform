# List every Cloud Identity Provider registry entry on the tenant (both
# Google and Microsoft Entra ID).
data "jamfplatform_pro_cloud_identity_providers" "all" {}

output "all_cloud_identity_providers" {
  value = data.jamfplatform_pro_cloud_identity_providers.all.cloud_identity_providers
}
