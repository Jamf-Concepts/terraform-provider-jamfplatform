# Copyright Jamf Software LLC 2026
# SPDX-License-Identifier: MPL-2.0

# Manage the Jamf Pro Jamf Teacher settings (Settings > Jamf apps > Jamf Teacher).
# Singleton — one record per tenant. Optional attributes you omit keep their
# current Jamf Pro value, including on the first apply (the resource adopts the
# existing settings); timezone must always be set.
resource "jamfplatform_pro_jamf_teacher_settings" "example" {
  # Allow limited management of students' devices by Jamf Teacher.
  enabled = true

  # IANA time zone the restriction times are evaluated in.
  timezone = "Europe/London"

  # Restrictions End Time — all restrictions set by Jamf Teacher are cleared
  # from student devices at this time (24-hour HH:MM:SS). Set to "" to clear.
  restrictions_end_time = "17:30:00"

  # Maximum Restriction Time, in seconds (the UI captures hours and minutes;
  # 28740 = 7 h 59 min, the UI maximum).
  maximum_restriction_time_seconds = 28740

  # Apps students can always use, even while restricted. Safelisting more than
  # one app prevents teachers from enabling Single App Mode; exactly one app
  # lets teachers lock student devices to it. Set to [] to clear.
  safelisted_apps = [
    {
      name      = "Example Calculator"
      bundle_id = "com.example.calculator"
    },
    {
      name      = "Example Notes"
      bundle_id = "com.example.notes"
    },
  ]
}
