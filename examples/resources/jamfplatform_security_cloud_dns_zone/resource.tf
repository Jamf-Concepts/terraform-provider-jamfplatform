# A custom DNS zone resolves hostnames under its domains through name servers you
# nominate, instead of public DNS. Adding one is a prerequisite for reaching
# enterprise apps on internal private networks over ZTNA.
resource "jamfplatform_security_cloud_dns_zone" "internal" {
  name = "Internal Services"

  # A wildcard covers only subdomains, so list the parent domain alongside it.
  domains = [
    "corp.example.com",
    "*.corp.example.com",
  ]

  # gateway_id names the gateway each name server is reachable through. Jamf
  # publishes a shared gateway per region — "Nearest Data Center" and the shared
  # IP pools in the admin UI — and your own ZTNA gateways may be used instead.
  # Each IP address may appear only once in a zone.
  name_servers = [
    {
      ip         = "203.0.113.53"
      gateway_id = "a7d2"
    },
    {
      ip         = "198.51.100.53"
      gateway_id = "1cbb"
    },
  ]
}
