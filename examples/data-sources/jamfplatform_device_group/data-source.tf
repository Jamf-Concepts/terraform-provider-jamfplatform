data "jamfplatform_device_group" "example_static_computer_group_by_name" {
  name        = "Example Static Computer Group"
  group_type  = "STATIC"
  device_type = "COMPUTER"
}

data "jamfplatform_device_group" "example_by_id" {
  id = "7f4cb2da-c7c4-44bd-a76b-147dd24eb613"
}

output "device_group_example_static_computer_group_by_name" {
  value = data.jamfplatform_device_group.example_static_computer_group_by_name
}

output "device_group_example_by_id" {
  value = data.jamfplatform_device_group.example_by_id
}
