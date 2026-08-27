# A grouped gateway is a routing and failover group over two or more of your own
# dedicated gateways. Every member must be the same form — all IPSec, or all
# internet.
resource "jamfplatform_security_cloud_ztna_grouped_gateway" "eu" {
  name = "EU Egress"

  # Order matters: it is the priority order the "First available" strategy walks,
  # and the admin UI shows it as a drag-to-reorder list.
  gateway_ids = [
    jamfplatform_security_cloud_ztna_gateway.london.id,
    jamfplatform_security_cloud_ztna_gateway.frankfurt.id,
  ]

  # "First available", "Random" or "Nearest".
  routing_strategy = "First available"

  # Required whatever the strategy, even though only "First available" uses it.
  required_gateway_stability = "30 minutes"

  tenant_ids = [var.security_cloud_tenant_id]
}

resource "jamfplatform_security_cloud_ztna_gateway" "london" {
  name          = "London Internet Egress"
  egress_region = "Europe - UK"
  tenant_ids    = [var.security_cloud_tenant_id]

  contact = {
    name  = "Network Operations"
    email = "netops@example.com"
  }
}

resource "jamfplatform_security_cloud_ztna_gateway" "frankfurt" {
  name          = "Frankfurt Internet Egress"
  egress_region = "Europe - Germany"
  tenant_ids    = [var.security_cloud_tenant_id]

  contact = {
    name  = "Network Operations"
    email = "netops@example.com"
  }
}

variable "security_cloud_tenant_id" {
  description = "Tenant granted access to these gateways."
  type        = string
}
