data "jamfplatform_pro_building" "example_by_id" {
  id = "12"
}

output "building_example_by_id" {
  value = data.jamfplatform_pro_building.example_by_id
}
