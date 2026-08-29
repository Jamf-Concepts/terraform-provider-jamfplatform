# Look up a device group by ID.
data "jamfplatform_security_cloud_device_group" "by_id" {
  id = "57497e81-d499-4f99-8fe8-8f262d0f5b8f"
}

# Or by name. Names are unique on the tenant but matched exactly, so the
# capitalisation has to be right.
data "jamfplatform_security_cloud_device_group" "executives" {
  name = "Executives"
}

output "executives_group_id" {
  value = data.jamfplatform_security_cloud_device_group.executives.id
}
