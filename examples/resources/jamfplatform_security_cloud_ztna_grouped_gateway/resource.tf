# A grouped gateway is a routing and failover group over two or more of your own
# dedicated gateways. Every member must be the same form — all IPSec, or all
# internet.
resource "jamfplatform_security_cloud_ztna_grouped_gateway" "eu" {
  name = "EU Egress"

  # Order matters: it is the priority order the FIRST_AVAILABLE (ACTIVE_STANDBY)
  # strategy walks, and the admin UI shows it as a drag-to-reorder list.
  gateway_ids = [
    jamfplatform_security_cloud_ztna_gateway.london.id,
    jamfplatform_security_cloud_ztna_gateway.frankfurt.id,
  ]

  # ACTIVE_STANDBY ("First available"), RANDOM ("Random") or NEAREST ("Nearest").
  routing_strategy = "ACTIVE_STANDBY"

  # "Required gateway stability". Required whatever the strategy, even though only
  # ACTIVE_STANDBY uses it.
  recovery_delay_seconds = 1800

  tenant_ids = [var.security_cloud_tenant_id]
}

resource "jamfplatform_security_cloud_ztna_gateway" "london" {
  name       = "London Internet Egress"
  datacenter = "eu-west-2"
  tenant_ids = [var.security_cloud_tenant_id]

  contact = {
    name  = "Network Operations"
    email = "netops@example.com"
  }
}

resource "jamfplatform_security_cloud_ztna_gateway" "frankfurt" {
  name       = "Frankfurt Internet Egress"
  datacenter = "eu-central-1"
  tenant_ids = [var.security_cloud_tenant_id]

  contact = {
    name  = "Network Operations"
    email = "netops@example.com"
  }
}

variable "security_cloud_tenant_id" {
  description = "Tenant granted access to these gateways."
  type        = string
}
