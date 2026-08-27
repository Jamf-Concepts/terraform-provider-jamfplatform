# Every dedicated gateway on the tenant. Jamf Security Cloud exposes no filter
# parameters for gateways, so narrow the result in Terraform.
data "jamfplatform_security_cloud_ztna_gateways" "all" {}

# Gateways whose tunnel is not up.
output "unhealthy_gateways" {
  value = [
    for gateway in data.jamfplatform_security_cloud_ztna_gateways.all.gateways :
    gateway.name if gateway.status.state != "UP"
  ]
}

# IPSec gateways, by egress region.
output "ipsec_gateways_by_region" {
  value = {
    for gateway in data.jamfplatform_security_cloud_ztna_gateways.all.gateways :
    gateway.name => gateway.datacenter if gateway.ipsec != null
  }
}
