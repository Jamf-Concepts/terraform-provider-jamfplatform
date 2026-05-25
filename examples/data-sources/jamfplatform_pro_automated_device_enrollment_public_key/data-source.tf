# The Jamf Pro Automated Device Enrollment (ADE) public key is a tenant-wide
# singleton. Upload the returned base64 key to Apple Business / School Manager
# when registering a new MDM server; download the resulting `.p7m` and supply
# it as `server_token` on `jamfplatform_pro_automated_device_enrollment`.
data "jamfplatform_pro_automated_device_enrollment_public_key" "this" {}

output "ade_public_key" {
  value     = data.jamfplatform_pro_automated_device_enrollment_public_key.this.public_key
  sensitive = true
}
