data "jamfplatform_inventory_mobile_device" "example" {
  sections = ["GENERAL", "APPLICATIONS"]                # Optional: Specify sections to include in the response
  id       = "6c1e1b1d172648827e6b4b7d874c3491348b38e6" # Replace with the actual mobile device UUID
}

output "device" {
  value = data.jamfplatform_inventory_mobile_device.example
}
