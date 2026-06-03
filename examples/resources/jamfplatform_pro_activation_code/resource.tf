resource "jamfplatform_pro_activation_code" "this" {
  organization_name = "Example Organization"

  # The activation code is a license secret. Source it from a variable or a secret
  # store rather than committing it; an invalid value can disable the tenant.
  code = var.jamf_pro_activation_code
}

variable "jamf_pro_activation_code" {
  type      = string
  sensitive = true
}
