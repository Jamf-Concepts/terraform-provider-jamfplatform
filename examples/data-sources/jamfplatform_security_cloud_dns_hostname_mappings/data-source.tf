# The tenant's hostname mappings. Takes no arguments, since there is one set per
# tenant. A tenant with no mappings reads back an empty collection rather than
# an error.
data "jamfplatform_security_cloud_dns_hostname_mappings" "current" {}

# Which host names are routed through ZTNA.
output "ztna_hostnames" {
  value = [
    for mapping in data.jamfplatform_security_cloud_dns_hostname_mappings.current.mappings :
    mapping.hostname if mapping.connect_to_ztna
  ]
}
