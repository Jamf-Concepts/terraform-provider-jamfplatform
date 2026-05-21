data "jamfplatform_device_group" "example_by_id" {
  id = "7f4cb2da-c7c4-44bd-a76b-147dd24eb613"
}

output "device_group_example_by_id" {
  value = data.jamfplatform_device_group.example_by_id
}

# `jamf_pro_id` is the numeric Jamf Pro classic ID resolved from the Pro
# /v2/groups endpoint. Use it from classic-API scope blocks (policies,
# configuration profiles, restricted software). Null when the API client lacks
# the `Read Groups` privilege.
output "device_group_example_jamf_pro_id" {
  value = data.jamfplatform_device_group.example_by_id.jamf_pro_id
}
