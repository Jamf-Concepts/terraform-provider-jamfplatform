data "jamfplatform_pro_mobile_device_extension_attribute" "by_id" {
  id = "1"
}

data "jamfplatform_pro_mobile_device_extension_attribute" "by_name" {
  name = "Device Role"
}

output "mobile_ea_input_type" {
  value = data.jamfplatform_pro_mobile_device_extension_attribute.by_name.input_type
}
