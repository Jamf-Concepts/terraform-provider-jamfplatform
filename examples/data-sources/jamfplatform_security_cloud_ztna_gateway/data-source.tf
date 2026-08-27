# Look up a dedicated gateway by ID.
data "jamfplatform_security_cloud_ztna_gateway" "by_id" {
  id = "a1b2"
}

# Or by name.
data "jamfplatform_security_cloud_ztna_gateway" "by_name" {
  name = "Frankfurt Private Apps"
}

# Point a custom DNS zone's name server at it.
resource "jamfplatform_security_cloud_dns_zone" "internal" {
  name    = "Internal Services"
  domains = ["corp.example.com"]

  authoritative_name_servers = [
    {
      ip_address = "10.100.0.53"
      gateway_id = data.jamfplatform_security_cloud_ztna_gateway.by_name.id
    },
  ]
}
