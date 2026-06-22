# Read the current Jamf Pro Impact Alert Notification settings.
data "jamfplatform_pro_impact_alert_notification_settings" "current" {}

output "deployable_objects_alert_enabled" {
  value = data.jamfplatform_pro_impact_alert_notification_settings.current.deployable_objects_alert_enabled
}
