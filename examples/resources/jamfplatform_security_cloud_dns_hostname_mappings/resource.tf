# Custom hostname mappings point internal host names at the addresses they should
# resolve to, so users reach internal resources while staying protected from mobile
# and network threats.
#
# This resource owns the tenant's ENTIRE set of mappings: a mapping added elsewhere
# and absent from this configuration is removed on the next apply. There is one set
# per tenant, so only one instance of this resource should exist in your
# configuration.
resource "jamfplatform_security_cloud_dns_hostname_mappings" "internal" {
  mappings = [
    # An IPv4-only mapping routed through ZTNA.
    {
      hostname              = "intranet.corp.example.com"
      ipv4_addresses        = ["10.1.0.10", "10.1.0.11"]
      connect_to_ztna       = true
      connect_to_secure_dns = false
    },

    # Dual-stack, and off both traffic vectoring paths.
    {
      hostname              = "wiki.corp.example.com"
      ipv4_addresses        = ["10.1.0.20"]
      ipv6_addresses        = ["2001:db8:1::20"]
      connect_to_ztna       = false
      connect_to_secure_dns = false
    },

    # IPv6 only. Omit an address list entirely when there is none. An empty
    # collection is not accepted, and every mapping needs at least one address.
    {
      hostname              = "build.corp.example.com"
      ipv6_addresses        = ["2001:db8:1::30"]
      connect_to_ztna       = false
      connect_to_secure_dns = true
    },
  ]
}
