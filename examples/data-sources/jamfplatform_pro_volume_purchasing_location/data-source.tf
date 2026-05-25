data "jamfplatform_pro_volume_purchasing_location" "example_by_id" {
  id = "1"
}

data "jamfplatform_pro_volume_purchasing_location" "example_by_name" {
  name = "vpp-prod"
}

output "vpp_example_by_id" {
  value = data.jamfplatform_pro_volume_purchasing_location.example_by_id
}

# Look up a specific content item's license availability by adam_id.
locals {
  microsoft_word_adam_id = "462058435"
}

output "vpp_microsoft_word_licenses_available" {
  value = one([
    for item in data.jamfplatform_pro_volume_purchasing_location.example_by_id.content :
    item.license_count_total - item.license_count_in_use
    if item.adam_id == local.microsoft_word_adam_id
  ])
}
