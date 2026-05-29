# Copyright Jamf Software LLC 2026
# SPDX-License-Identifier: MPL-2.0

# Read the current Jamf Pro Re-enrollment settings.
data "jamfplatform_pro_re_enrollment_settings" "this" {}

output "clear_management_history" {
  value = data.jamfplatform_pro_re_enrollment_settings.this.clear_management_history
}
