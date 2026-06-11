# Copyright Jamf Software LLC 2026
# SPDX-License-Identifier: MPL-2.0

# An SMB file share distribution point with read/write and read-only accounts
# and HTTPS downloads enabled.
resource "jamfplatform_pro_file_share_distribution_point" "primary" {
  name                         = "Main DP"
  server_name                  = "dp.example.com"
  file_sharing_connection_type = "SMB"
  share_name                   = "CasperShare"
  port                         = 445
  workgroup                    = "WORKGROUP"

  # Read/Write account. The password is WriteOnly — bump
  # read_write_password_wo_version to rotate it.
  read_write_username            = "casperadmin"
  read_write_password            = sensitive("change-me-rw")
  read_write_password_wo_version = 1

  # Read-only account.
  read_only_username            = "casperinstall"
  read_only_password            = sensitive("change-me-ro")
  read_only_password_wo_version = 1

  # HTTPS downloads with username/password authentication.
  https_enabled             = true
  https_port                = 443
  https_context             = "casper"
  https_security_type       = "USERNAME_PASSWORD"
  https_username            = "httpsuser"
  https_password            = sensitive("change-me-https")
  https_password_wo_version = 1
}

# A second distribution point that fails over to the first and randomly shares
# load between them.
resource "jamfplatform_pro_file_share_distribution_point" "failover" {
  name                         = "Failover DP"
  server_name                  = "dp2.example.com"
  file_sharing_connection_type = "SMB"
  share_name                   = "CasperShare"

  read_write_username            = "casperadmin"
  read_write_password            = sensitive("change-me-rw")
  read_write_password_wo_version = 1
  read_only_username             = "casperinstall"
  read_only_password             = sensitive("change-me-ro")
  read_only_password_wo_version  = 1

  backup_distribution_point_id = jamfplatform_pro_file_share_distribution_point.primary.id
  enable_load_balancing        = true
}
