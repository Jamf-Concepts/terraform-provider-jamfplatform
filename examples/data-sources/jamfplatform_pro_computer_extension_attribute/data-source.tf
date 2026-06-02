data "jamfplatform_pro_computer_extension_attribute" "by_id" {
  id = "1"
}

data "jamfplatform_pro_computer_extension_attribute" "by_name" {
  name = "Asset Tag"
}

output "computer_ea_input_type" {
  value = data.jamfplatform_pro_computer_extension_attribute.by_name.input_type
}
