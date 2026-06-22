# Copyright Jamf Software LLC 2026
# SPDX-License-Identifier: MPL-2.0

# Manage the Jamf Pro Re-enrollment settings (Settings > Global > Re-enrollment).
# Singleton — one record per tenant. These options decide which data Jamf Pro
# clears from a computer or mobile device when it re-enrolls.
resource "jamfplatform_pro_re_enrollment_settings" "this" {
  clear_policy_logs                  = true
  clear_location_information         = true
  clear_location_information_history = false
  clear_extension_attributes         = true
  clear_software_update_plans        = false

  # Clear nothing / failed commands / pending and failed / everything.
  clear_management_history = "DELETE_EVERYTHING_EXCEPT_ACKNOWLEDGED"
}
