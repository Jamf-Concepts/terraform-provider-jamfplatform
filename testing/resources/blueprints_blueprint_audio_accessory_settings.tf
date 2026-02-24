resource "jamfplatform_blueprints_blueprint" "test_audio_accessory_settings" {
  name        = "Terraform Test Audio Accessory Settings ${var.test_id}"
  description = "Managed by Terraform"
  deployed    = false

  device_groups = [jamfplatform_device_group.test_smart_computer.id]

  audio_accessory_settings = {
    temporary_pairing_disabled = false
    unpairing_time_policy      = "Hour"
    unpairing_time_hour        = 22
  }
}
