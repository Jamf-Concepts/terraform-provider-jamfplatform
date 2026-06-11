# Read the current macOS Onboarding configuration. Items are returned in priority order.
data "jamfplatform_pro_macos_onboarding" "current" {}

output "onboarding_enabled" {
  value = data.jamfplatform_pro_macos_onboarding.current.enabled
}

output "onboarding_item_names" {
  value = [for item in data.jamfplatform_pro_macos_onboarding.current.onboarding_items : item.entity_name]
}
