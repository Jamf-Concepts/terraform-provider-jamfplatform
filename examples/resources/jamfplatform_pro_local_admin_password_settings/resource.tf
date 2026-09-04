# Copyright Jamf Software LLC 2026
# SPDX-License-Identifier: MPL-2.0

# Manage the Jamf Pro local administrator password (LAPS) settings
# (Settings > Computer Management > Security > "Password settings for managed
# local administrator accounts"). One record per tenant.
resource "jamfplatform_pro_local_admin_password_settings" "this" {
  # Enable LAPS for managed local administrator accounts created via PreStage enrollment.
  laps_for_prestage_accounts_enabled = true

  # Automatically rotate passwords every 7 days (set to "Never" to turn rotation off).
  rotation_interval = "7 days"

  # Rotate a password 1 hour after it is viewed in the inventory record.
  rotation_after_viewing_interval = "1 hour"
}
