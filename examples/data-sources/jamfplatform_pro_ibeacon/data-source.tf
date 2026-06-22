data "jamfplatform_pro_ibeacon" "example_by_id" {
  id = "3"
}

data "jamfplatform_pro_ibeacon" "example_by_name" {
  name = "Reception"
}

output "ibeacon_example_by_id" {
  value = data.jamfplatform_pro_ibeacon.example_by_id
}

output "ibeacon_example_by_name" {
  value = data.jamfplatform_pro_ibeacon.example_by_name
}
