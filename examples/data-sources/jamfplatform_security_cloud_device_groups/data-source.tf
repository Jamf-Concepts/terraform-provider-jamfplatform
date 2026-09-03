# Every device group on the tenant. Jamf Security Cloud exposes no filter
# parameters for groups, so narrow the result in Terraform.
data "jamfplatform_security_cloud_device_groups" "all" {}

output "group_names" {
  value = [for group in data.jamfplatform_security_cloud_device_groups.all.device_groups : group.name]
}

# The built-in group is included, and it is the one entry with no id, so nothing
# can reference it. Filter it out before mapping names to IDs.
output "group_ids_by_name" {
  value = {
    for group in data.jamfplatform_security_cloud_device_groups.all.device_groups :
    group.name => group.id if !group.built_in
  }
}
