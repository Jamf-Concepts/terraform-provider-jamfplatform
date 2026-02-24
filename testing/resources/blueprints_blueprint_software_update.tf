resource "jamfplatform_blueprints_blueprint" "test_software_update" {
  name        = "Terraform Test Software Update ${var.test_id}"
  description = "Managed by Terraform"
  deployed    = false

  device_groups = [jamfplatform_device_group.test_smart_computer.id]

  software_update = {
    ignore_major_versions = true
    deployment_time       = "17:00"
    enforce_after_days    = 7
  }
}
