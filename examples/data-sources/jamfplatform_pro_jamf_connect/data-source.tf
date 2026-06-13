# Look up the Jamf Connect deployment settings for a configuration profile.
# Supply exactly one of config_profile_uuid, profile_id, or profile_name.

# By the configuration profile's Jamf Pro ID.
data "jamfplatform_pro_jamf_connect" "by_id" {
  profile_id = 47
}

# By the configuration profile's display name (exact match).
data "jamfplatform_pro_jamf_connect" "by_name" {
  profile_name = "Jamf Connect Login"
}

output "connect_deployment_type" {
  value = data.jamfplatform_pro_jamf_connect.by_id.auto_deployment_type
}
