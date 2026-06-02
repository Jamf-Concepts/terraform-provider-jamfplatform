data "jamfplatform_pro_user_extension_attribute" "by_id" {
  id = "1"
}

data "jamfplatform_pro_user_extension_attribute" "by_name" {
  name = "Employee ID"
}

output "user_ea_data_type" {
  value = data.jamfplatform_pro_user_extension_attribute.by_name.data_type
}
