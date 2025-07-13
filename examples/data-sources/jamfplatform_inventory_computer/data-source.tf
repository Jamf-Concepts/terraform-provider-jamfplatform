data "jamfplatform_inventory_computer" "example" {
  id = "C46BD329-29FE-52DC-92E3-B397C0E22199" # Replace with the actual computer UUID
}

output "computer" {
  value = data.jamfplatform_inventory_computer.example
}
