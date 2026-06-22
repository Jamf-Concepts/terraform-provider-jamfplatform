# Look up by ID.
data "jamfplatform_pro_mobile_device_configuration_profile" "by_id" {
  id = "42"
}

# Look up by exact name.
data "jamfplatform_pro_mobile_device_configuration_profile" "by_name" {
  name = "Restrictions – Block App Store"
}

output "category_name" {
  value = data.jamfplatform_pro_mobile_device_configuration_profile.by_id.general.category_name
}
