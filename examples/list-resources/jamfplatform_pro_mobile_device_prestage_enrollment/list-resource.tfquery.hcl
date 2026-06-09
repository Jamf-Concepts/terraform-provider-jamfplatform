# Copyright Jamf Software LLC 2026
# SPDX-License-Identifier: MPL-2.0

list "jamfplatform_pro_mobile_device_prestage_enrollment" "all" {
  provider = jamfplatform

  filter = {
    name_substring = "ipad"
  }
}
