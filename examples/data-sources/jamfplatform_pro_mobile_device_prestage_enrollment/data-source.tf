# Copyright Jamf Software LLC 2026
# SPDX-License-Identifier: MPL-2.0

data "jamfplatform_pro_mobile_device_prestage_enrollment" "by_id" {
  id = "1"
}

data "jamfplatform_pro_mobile_device_prestage_enrollment" "by_name" {
  name = "Standard iPad PreStage"
}

output "prestage_uuid" {
  value = data.jamfplatform_pro_mobile_device_prestage_enrollment.by_name.profile_uuid
}
