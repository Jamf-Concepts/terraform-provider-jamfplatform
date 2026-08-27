# The Jamf-managed shared gateway catalogue: one "Nearest Data Center" entry plus a
# shared IP pool per region. Read-only, and the same for every entitled tenant.
data "jamfplatform_security_cloud_ztna_shared_gateways" "all" {}

# Resolve a shared gateway by name so a custom DNS zone need not hard-code the
# four-character ID.
locals {
  nearest_data_center = one([
    for gateway in data.jamfplatform_security_cloud_ztna_shared_gateways.all.shared_gateways :
    gateway.id if gateway.name == "Nearest Data Center"
  ])
}

resource "jamfplatform_security_cloud_dns_zone" "internal" {
  name    = "Internal Services"
  domains = ["corp.example.com", "*.corp.example.com"]

  authoritative_name_servers = [
    {
      ip_address = "203.0.113.53"
      gateway_id = local.nearest_data_center
    },
  ]
}

output "shared_gateway_ids" {
  value = {
    for gateway in data.jamfplatform_security_cloud_ztna_shared_gateways.all.shared_gateways :
    gateway.name => gateway.id
  }
}
