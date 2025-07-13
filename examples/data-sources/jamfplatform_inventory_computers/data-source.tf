data "jamfplatform_inventory_computers" "by_name" {
  filter = "general.name=='My Computer'"
}

output "computers_by_name" {
  value = data.jamfplatform_inventory_computers.by_name
}

data "jamfplatform_inventory_computers" "all" {}

output "all_computers" {
  value = data.jamfplatform_inventory_computers.all
}
