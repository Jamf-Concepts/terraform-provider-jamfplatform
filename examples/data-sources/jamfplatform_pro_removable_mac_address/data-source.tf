data "jamfplatform_pro_removable_mac_address" "example_by_id" {
  id = "3"
}

data "jamfplatform_pro_removable_mac_address" "example_by_mac_address" {
  mac_address = "00:A0:C9:14:C8:20"
}

output "removable_mac_address_example_by_id" {
  value = data.jamfplatform_pro_removable_mac_address.example_by_id
}

output "removable_mac_address_example_by_mac_address" {
  value = data.jamfplatform_pro_removable_mac_address.example_by_mac_address
}
