resource "jamfplatform_blueprints_blueprint" "test_legacy_payloads" {
  name        = "Terraform Test Legacy Payloads ${var.test_id}"
  description = "Managed by Terraform"
  deployed    = false

  device_groups = [jamfplatform_device_group.test_smart_computer.id]

  legacy_payloads = jsonencode([
    {
      allowSafariHistoryClearing = false
      allowSafariPrivateBrowsing = false
      payloadType                = "com.apple.applicationaccess"
      payloadIdentifier          = "da0ac44c-419e-43ff-b300-00b0e645fa7e"
    }
  ])
}
