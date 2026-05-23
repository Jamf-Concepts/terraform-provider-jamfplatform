data "jamfplatform_pro_printer" "example_by_id" {
  id = "67"
}

data "jamfplatform_pro_printer" "example_by_name" {
  name = "Front Desk"
}

output "printer_example_by_id" {
  value = data.jamfplatform_pro_printer.example_by_id
}

output "printer_example_by_name" {
  value = data.jamfplatform_pro_printer.example_by_name
}
