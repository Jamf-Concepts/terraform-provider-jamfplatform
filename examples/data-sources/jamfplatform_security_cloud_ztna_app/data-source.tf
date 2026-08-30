# Look an access policy application up by ID.
data "jamfplatform_security_cloud_ztna_app" "by_id" {
  id = "27f04387-0a12-4f70-9256-eeccc67d7304"
}

# Or by name. Application names are not required to be unique, so a name matching
# more than one application is an error — use the ID in that case.
data "jamfplatform_security_cloud_ztna_app" "by_name" {
  name = "Internal CRM"
}

# A predefined application has no name of its own, so look one up by the Jamf
# definition it is based on.
data "jamfplatform_security_cloud_ztna_predefined_apps" "all" {}

data "jamfplatform_security_cloud_ztna_app" "slack" {
  predefined_app_id = one([
    for app in data.jamfplatform_security_cloud_ztna_predefined_apps.all.predefined_apps :
    app.id if app.name == "Slack"
  ])
}

output "crm_routing_gateway" {
  value = data.jamfplatform_security_cloud_ztna_app.by_id.routing.gateway_id
}
