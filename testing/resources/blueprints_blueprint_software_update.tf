resource "jamfplatform_blueprints_blueprint" "test_software_update" {
  name        = "Terraform Test Software Update ${var.test_id}"
  description = "Managed by Terraform"
  deployed    = false

  device_groups = [jamfplatform_device_group.test_smart_computer.id]

  software_update {
    target_os_version      = "26.0.1"
    target_local_date_time = "2025-10-10T12:00:00"
    details_url_value      = "https://soundmacguy.wordpress.com"
  }
}
