resource "jamfplatform_blueprints_blueprint" "test_disk_management" {
  name        = "Terraform Test Disk Management ${var.test_id}"
  description = "Managed by Terraform"
  deployed    = false

  device_groups = [jamfplatform_device_group.test_smart_computer.id]

  disk_management_settings = {
    external_storage = "ReadOnly"
    network_storage  = "Disallowed"
  }
}
