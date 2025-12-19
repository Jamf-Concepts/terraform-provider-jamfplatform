data "jamfplatform_device_group" "example_by_id" {
  id = "7f4cb2da-c7c4-44bd-a76b-147dd24eb613"
}

output "device_group_example_by_id" {
  value = data.jamfplatform_device_group.example_by_id
}
