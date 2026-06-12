# A supervision identity is the certificate used to supervise and enroll devices
# through Apple Configurator (Settings > Apple Configurator Enrollment).
#
# The password and certificate are write-only: they are sent to Jamf Pro but never
# stored in Terraform state, and Jamf Pro never returns them. Only display_name can
# be changed in place; changing the password or certificate replaces the identity
# (terraform apply -replace).

# Generate path: omit certificate_data and Jamf Pro mints a new identity for you.
resource "jamfplatform_pro_supervision_identity" "generated" {
  display_name = "Apple Configurator Identity"
  password     = var.supervision_identity_password
}

# Import path: supply certificate_data to import an existing .p12 identity.
resource "jamfplatform_pro_supervision_identity" "imported" {
  display_name = "Imported Configurator Identity"
  password     = var.supervision_identity_password

  # Supply the .p12 with filebase64() so its bytes never appear in configuration.
  certificate_data = filebase64("${path.module}/identity.p12")
}

variable "supervision_identity_password" {
  type      = string
  sensitive = true
}

output "generated_identity_common_name" {
  value = jamfplatform_pro_supervision_identity.generated.common_name
}

output "generated_identity_expiration" {
  value = jamfplatform_pro_supervision_identity.generated.expiration_date
}
