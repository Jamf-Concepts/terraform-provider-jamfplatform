# macOS Onboarding (Settings > Self Service > macOS Onboarding) — a curated, ordered
# list of Self Service items (policies, configuration profiles, apps) presented to
# users during macOS onboarding. Singleton: one configuration per tenant.
#
# Item order is significant: items appear to users in `onboarding_items` order, and
# `priority` is derived from that order automatically. The list fully replaces what is
# stored — declare the complete set; removing an item here removes it from onboarding.
#
# Each referenced object must be enabled and available in Self Service. Discover
# eligible IDs with the jamfplatform_pro_macos_onboarding_eligible_items data source.
resource "jamfplatform_pro_macos_onboarding" "example" {
  enabled = true

  onboarding_items = [
    {
      # A Self Service policy.
      entity_id                = "132"
      self_service_entity_type = "OS_X_POLICY"
    },
    {
      # A configuration profile.
      entity_id                = "11"
      self_service_entity_type = "OS_X_CONFIG_PROFILE"
    },
    {
      # A Mac App Store app.
      entity_id                = "87"
      self_service_entity_type = "OS_X_MAC_APP"
    },
  ]
}
