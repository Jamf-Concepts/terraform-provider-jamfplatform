# Copyright Jamf Software LLC 2026
# SPDX-License-Identifier: MPL-2.0

# Read the current Jamf Pro local administrator password (LAPS) settings.
data "jamfplatform_pro_local_admin_password_settings" "this" {}

output "rotation_interval" {
  value = data.jamfplatform_pro_local_admin_password_settings.this.rotation_interval
}
