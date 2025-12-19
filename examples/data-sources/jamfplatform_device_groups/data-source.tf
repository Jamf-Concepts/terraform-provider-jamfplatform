data "jamfplatform_device_groups" "static_computer_groups" {
  group_type  = "static"
  device_type = "computer"
}

output "device_groups_static_computer_groups" {
  value = data.jamfplatform_device_groups.static_computer_groups
}
