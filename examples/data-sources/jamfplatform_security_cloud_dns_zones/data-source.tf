# Every custom DNS zone on the tenant. Jamf Security Cloud exposes no filter
# parameters for zones, so narrow the result in Terraform.
data "jamfplatform_security_cloud_dns_zones" "all" {}

output "zone_names" {
  value = [for zone in data.jamfplatform_security_cloud_dns_zones.all.dns_zones : zone.name]
}

# Zones that serve a wildcard domain.
output "wildcard_zones" {
  value = [
    for zone in data.jamfplatform_security_cloud_dns_zones.all.dns_zones :
    zone.name if length([for domain in zone.domains : domain if startswith(domain, "*.")]) > 0
  ]
}
