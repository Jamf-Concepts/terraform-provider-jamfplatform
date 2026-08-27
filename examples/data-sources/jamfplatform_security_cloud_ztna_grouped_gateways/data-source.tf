# Every grouped gateway on the tenant.
data "jamfplatform_security_cloud_ztna_grouped_gateways" "all" {}

output "grouped_gateway_strategies" {
  value = {
    for group in data.jamfplatform_security_cloud_ztna_grouped_gateways.all.grouped_gateways :
    group.name => group.routing_strategy
  }
}
