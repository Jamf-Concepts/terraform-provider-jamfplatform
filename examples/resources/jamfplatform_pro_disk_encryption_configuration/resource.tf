# Individual key_type: each Mac that enables FileVault generates its own
# personal recovery key. No institutional certificate required.
resource "jamfplatform_pro_disk_encryption_configuration" "individual" {
  name                     = "Individual recovery key"
  key_type                 = "Individual"
  file_vault_enabled_users = "Current or Next User"
}

# Institutional key_type: Jamf issues recovery keys derived from the
# uploaded PKCS12 certificate. The `data` payload is the base64 of the
# `.p12` file contents; `password` is the import password. The plaintext
# `password` is a Terraform `WriteOnly` attribute, sent on writes but
# never persisted in state. Pair with `password_wo_version` to rotate
# the stored password (bump the integer to force the next apply to re-send
# the certificate). `certificate_type` and `key` (Subject DN) are returned
# by Jamf Pro and not user-settable beyond the create-time values supplied
# here.
resource "jamfplatform_pro_disk_encryption_configuration" "institutional" {
  name                     = "Institutional recovery key"
  key_type                 = "Institutional"
  file_vault_enabled_users = "Current or Next User"

  institutional_recovery_key = {
    # `certificate_type` is required whenever an IRK block is supplied.
    # Use "PKCS12" for .p12 uploads, "DER" for .cer binary, or "PEM" for
    # .pem text.
    certificate_type = "PKCS12"
    # Base64 of your `.p12` file. Replace with `filebase64("./irk.p12")`
    # or the contents of a Vault-backed secret.
    data     = "BASE64_OF_YOUR_PKCS12_FILE_GOES_HERE=="
    password = sensitive("change-me")
  }
}

# Individual + Institutional: a per-Mac personal key AND a recovery key
# derived from the uploaded cert. Same upload shape as the Institutional
# example. `key_type` is case-sensitive, so use the exact string
# `Individual and Institutional` (lowercase `and`).
resource "jamfplatform_pro_disk_encryption_configuration" "both" {
  name                     = "Individual and Institutional"
  key_type                 = "Individual and Institutional"
  file_vault_enabled_users = "Management Account"

  institutional_recovery_key = {
    certificate_type = "PKCS12"
    data             = "BASE64_OF_YOUR_PKCS12_FILE_GOES_HERE=="
    password         = sensitive("change-me")
  }
}
