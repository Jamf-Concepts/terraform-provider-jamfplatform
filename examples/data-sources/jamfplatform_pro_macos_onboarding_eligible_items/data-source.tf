# Discover the Self Service objects eligible to be referenced from
# jamfplatform_pro_macos_onboarding.onboarding_items, for a given entity type.
# entity_type is one of: policies, configuration_profiles, apps.
data "jamfplatform_pro_macos_onboarding_eligible_items" "policies" {
  entity_type = "policies"
}

# Each item's `id` is used as `entity_id`, paired with the matching
# self_service_entity_type (policies -> OS_X_POLICY).
output "eligible_policy_ids" {
  value = [for item in data.jamfplatform_pro_macos_onboarding_eligible_items.policies.items : item.id]
}
