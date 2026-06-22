# Manages a Jamf Pro Automated Device Enrollment (ADE) instance.
#
# `server_token` is `WriteOnly` — the base64-encoded contents of the `.p7m`
# server token downloaded from Apple Business Manager / Apple School Manager
# are sent to Jamf Pro on writes but never persisted in Terraform state.
# Bump `server_token_wo_version` to rotate the stored token on the next apply
# (triggers `ReplaceDeviceEnrollmentTokenV1`).
#
# The token is generated against a tenant-specific MDM public key. Pull the
# tenant's key with the companion data source
# `jamfplatform_pro_automated_device_enrollment_public_key`, upload it to
# Apple Business / School Manager, then download the `.p7m` Apple generates.
resource "jamfplatform_pro_automated_device_enrollment" "prod" {
  name                    = "ade-prod"
  server_token            = filebase64("${path.module}/tokens/ade-prod.p7m")
  server_token_wo_version = 1
  token_file_name         = "ade-prod.p7m"

  # site_id and supervision_identity_id are optional. Omit to let Jamf Pro
  # decide; the server emits the sentinel "-1" when unset.
  # site_id                 = "1"
  # supervision_identity_id = "1"
}

output "ade_prod_id" {
  value = jamfplatform_pro_automated_device_enrollment.prod.id
}

output "ade_prod_token_expiration" {
  value = jamfplatform_pro_automated_device_enrollment.prod.token_expiration_date
}
