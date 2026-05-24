# Look up by ID.
data "jamfplatform_pro_macos_configuration_profile" "by_id" {
  id = "42"
}

# Look up by exact name.
data "jamfplatform_pro_macos_configuration_profile" "by_name" {
  name = "PPPC – Accessibility"
}

output "category_name" {
  value = data.jamfplatform_pro_macos_configuration_profile.by_id.category_name
}
