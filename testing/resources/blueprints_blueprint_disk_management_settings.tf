resource "jamfplatform_blueprints_blueprint" "test_disk_management" {
  name        = "Terraform Test Disk Management ${var.test_id}"
  description = "Managed by Terraform"

  device_groups = [data.jamfpro_group.test_target_computer_group.group_platform_id]

  disk_management_settings {
    external_storage = "ReadOnly"
    network_storage  = "Disallowed"
  }
}
