# Look up a custom DNS zone by ID.
data "jamfplatform_security_cloud_dns_zone" "by_id" {
  id = "f5734162-26d4-4d0f-a3ae-62f952b3688f"
}

# Or by name. Zone names are not required to be unique, so a name matching more
# than one zone is an error — look those up by ID.
data "jamfplatform_security_cloud_dns_zone" "by_name" {
  name = "Internal Services"
}

output "internal_services_domains" {
  value = data.jamfplatform_security_cloud_dns_zone.by_name.domains
}
