resource "jamfplatform_blueprints_blueprint" "test_legacy_payloads" {
  name        = "Terraform Test Legacy Payloads ${var.test_id}"
  description = "Managed by Terraform"
  deployed    = false

  device_groups = [jamfplatform_device_group.test_smart_computer.id]

  legacy_payloads = [
    {
      payload_type = "com.apple.applicationaccess"
      settings = {
        allowSafariHistoryClearing = false
        allowSafariPrivateBrowsing = false
      }
    }
  ]
}
