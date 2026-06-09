# Jamf Pro allows at most one JSON Web Token configuration per instance.
# The encryption key is WriteOnly — bump encryption_key_wo_version to rotate it.
resource "jamfplatform_pro_pki_json_web_token_configuration" "example" {
  name                      = "Jamf Setup token"
  encryption_key_wo         = var.jwt_encryption_key # WriteOnly
  encryption_key_wo_version = 1
  token_expiry              = 30 # minutes (1-120)
  enabled                   = true
}

variable "jwt_encryption_key" {
  type      = string
  sensitive = true
  default   = null
}
