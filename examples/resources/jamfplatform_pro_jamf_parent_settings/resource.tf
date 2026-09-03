# Copyright Jamf Software LLC 2026
# SPDX-License-Identifier: MPL-2.0

# Manage the Jamf Pro Jamf Parent settings (Settings > Jamf apps > Jamf Parent).
# One record per tenant. Optional attributes you omit keep their current Jamf
# Pro value, including on the first apply (the resource adopts the existing
# settings); timezone, device_group_id and restricted_times must always be set.
resource "jamfplatform_pro_jamf_parent_settings" "example" {
  # Allow limited management of students' devices by parents or guardians
  # using Jamf Parent.
  enabled = true

  # IANA time zone the restricted times are evaluated in.
  timezone = "Europe/London"

  # Student Device Group: id of the mobile device group (smart or static)
  # whose members Jamf Parent can manage. Reference your own group resource or
  # data source instead of a literal id where possible.
  device_group_id = 1

  # Jamf Parent App Restrictions: per-day Start/End times, keyed by uppercase
  # day name. Only the days you declare are sent and stored; use {} for no
  # restrictions.
  restricted_times = {
    MONDAY = {
      begin_time = "08:30:00"
      end_time   = "15:30:00"
    }
    FRIDAY = {
      begin_time = "09:00:00"
      end_time   = "14:00:00"
    }
  }

  # Allow Jamf Parent to Clear Student Device Passcode.
  allow_clear_passcode = true

  # Revoke Jamf Parent management capabilities when wiping or re-enrolling.
  revoke_on_wipe_and_re_enroll = true

  # Apps students can always use, even while restricted. Set to [] to clear.
  safelisted_apps = [
    {
      name      = "Example Calculator"
      bundle_id = "com.example.calculator"
    },
  ]
}
