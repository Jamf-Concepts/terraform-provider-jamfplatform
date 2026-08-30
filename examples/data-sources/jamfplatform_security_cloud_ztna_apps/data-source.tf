# Lists every access policy application on the tenant. Jamf Security Cloud exposes no
# filter parameters, so filter the result in Terraform.
data "jamfplatform_security_cloud_ztna_apps" "all" {}

# Applications still routing traffic directly, which reach their servers without the
# ZTNA tunnel.
output "direct_routed_applications" {
  value = [
    for app in data.jamfplatform_security_cloud_ztna_apps.all.ztna_apps :
    coalesce(app.name, app.predefined_app_id)
    if app.routing.traffic_routing == "Direct device routing"
  ]
}

# Applications any device in the fleet can reach.
output "unrestricted_applications" {
  value = [
    for app in data.jamfplatform_security_cloud_ztna_apps.all.ztna_apps :
    coalesce(app.name, app.predefined_app_id) if app.all_device_groups
  ]
}
