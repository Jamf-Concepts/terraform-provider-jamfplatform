resource "jamfplatform_pro_impact_alert_notification_settings" "this" {
  # Deployable objects (policies, configuration profiles, apps, managed software updates)
  deployable_objects_alert_enabled             = true
  deployable_objects_confirmation_code_enabled = true

  # Scopeable objects (smart groups, static groups, classes)
  scopeable_objects_alert_enabled             = true
  scopeable_objects_confirmation_code_enabled = false
}
