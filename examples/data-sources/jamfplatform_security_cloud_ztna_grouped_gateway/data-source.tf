# Look up a grouped gateway by ID.
data "jamfplatform_security_cloud_ztna_grouped_gateway" "by_id" {
  id = "3fa85f64-5717-4562-b3fc-2c963f66afa6"
}

# Or by name.
data "jamfplatform_security_cloud_ztna_grouped_gateway" "by_name" {
  name = "EU Egress"
}

output "eu_egress_members" {
  value = data.jamfplatform_security_cloud_ztna_grouped_gateway.by_name.gateway_ids
}
