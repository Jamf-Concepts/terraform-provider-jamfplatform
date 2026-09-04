# A Volume Purchasing notification (Settings → Volume purchasing → Notifications).
# Emails the chosen Jamf Pro accounts and external recipients a daily summary
# when one of the selected events occurs.
resource "jamfplatform_pro_volume_purchasing_notification" "low_licenses" {
  name    = "Volume Purchasing — Low Licenses"
  enabled = true

  # Events that send the notification:
  #   NO_MORE_LICENSES:       a location runs out of licenses
  #   REMOVED_FROM_APP_STORE: an item is removed from the App Store
  triggers = ["NO_MORE_LICENSES", "REMOVED_FROM_APP_STORE"]

  # Volume Purchasing location IDs the notification covers
  # (jamfplatform_pro_volume_purchasing_location). Omit or set to [] for none.
  location_ids = ["3"]

  # Jamf Pro account IDs that receive the daily summary email.
  internal_recipients = ["66", "67"]

  # Email addresses outside Jamf Pro that receive the daily summary.
  external_recipients = [
    { email = "vpp-admin@example.com", name = "VPP Admin" },
  ]
}
