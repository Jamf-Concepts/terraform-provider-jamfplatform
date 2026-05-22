data "jamfplatform_pro_dock_item" "example_by_id" {
  id = "3"
}

data "jamfplatform_pro_dock_item" "example_by_name" {
  name = "Calculator"
}

output "dock_item_example_by_id" {
  value = data.jamfplatform_pro_dock_item.example_by_id
}

output "dock_item_example_by_name" {
  value = data.jamfplatform_pro_dock_item.example_by_name
}
