# Look up a Cloud Identity Provider registry entry by ID or by display name.
# Covers both Google and Microsoft Entra ID providers. Exactly one of `id` or
# `display_name` must be supplied.
data "jamfplatform_pro_cloud_identity_provider" "by_id" {
  id = "1"
}

data "jamfplatform_pro_cloud_identity_provider" "by_name" {
  display_name = "Google Workspace"
}

output "cloud_identity_provider_by_id" {
  value = data.jamfplatform_pro_cloud_identity_provider.by_id
}

output "cloud_identity_provider_by_name" {
  value = data.jamfplatform_pro_cloud_identity_provider.by_name
}
