# Copyright Jamf Software LLC 2026
# SPDX-License-Identifier: MPL-2.0

data "jamfplatform_pro_file_share_distribution_point" "example_by_id" {
  id = "1"
}

data "jamfplatform_pro_file_share_distribution_point" "example_by_name" {
  name = "Main DP"
}

output "file_share_distribution_point_by_id" {
  value = data.jamfplatform_pro_file_share_distribution_point.example_by_id
}

output "file_share_distribution_point_by_name" {
  value = data.jamfplatform_pro_file_share_distribution_point.example_by_name
}
