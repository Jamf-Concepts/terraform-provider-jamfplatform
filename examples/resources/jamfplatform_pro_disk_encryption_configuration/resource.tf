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
# password is write-only — only the server-side redaction sentinel
# (literal `********************`) is returned on reads. The
# `certificate_type` and `key` (Subject DN) are server-derived.
resource "jamfplatform_pro_disk_encryption_configuration" "institutional" {
  name                     = "Institutional recovery key"
  key_type                 = "Institutional"
  file_vault_enabled_users = "Current or Next User"

  institutional_recovery_key = {
    # Base64 of your `.p12` file. Replace with `filebase64("./irk.p12")`
    # or the contents of a Vault-backed secret.
    data     = "BASE64_OF_YOUR_PKCS12_FILE_GOES_HERE=="
    password = sensitive("change-me")
  }
}

# Individual + Institutional: a per-Mac personal key AND a recovery key
# derived from the uploaded cert. Same upload shape as the Institutional
# example. Note that the user types `Individual And Institutional`
# (Title Case) — the provider canonicalises it to the wire form
# `Individual and Institutional` (lowercase `and`) so plan output
# matches what the Jamf Pro server returns on read.
resource "jamfplatform_pro_disk_encryption_configuration" "both" {
  name                     = "Individual and Institutional"
  key_type                 = "Individual And Institutional"
  file_vault_enabled_users = "Management Account"

  institutional_recovery_key = {
    data     = "BASE64_OF_YOUR_PKCS12_FILE_GOES_HERE=="
    password = sensitive("change-me")
  }
}
