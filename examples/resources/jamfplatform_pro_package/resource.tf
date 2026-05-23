# JCDS mode: local-path upload. The provider streams the binary to the
# cloud distribution point, then polls until JCDS has computed every hash
# and `cloud_transfer_status` becomes "READY". Hash attributes are
# Computed in this mode — supplying any of them in config is a plan-time
# error (`ConflictsWith(package_file_source)`).
resource "jamfplatform_pro_package" "jcds_local" {
  display_name        = "MyApp 1.0.0"
  file_name           = "MyApp-1.0.0.pkg"
  category_id         = "-1"
  info                = "Internal build of MyApp"
  reboot_required     = false
  priority            = 10
  package_file_source = "/path/to/MyApp-1.0.0.pkg"

  # Optional integrity hint: validated against the locally-computed
  # SHA-3-512 before any bytes leave the workstation. A mismatch errors
  # out and skips the upload — useful for guarding against on-disk
  # corruption.
  package_file_source_checksum = "0123456789abcdef..." # hex sha3-512

  # JCDS uploads can take minutes for multi-GB binaries. The verification
  # poll inherits this deadline through context.
  timeouts {
    create = "30m"
    update = "30m"
  }
}

# JCDS mode: HTTPS URL upload. The provider downloads the URL into a
# sanitised tempfile (8 GiB cap, 10-redirect cap, filename resolved
# post-redirect), hashes it, uploads it, then runs the same verification
# poll as the local-path example.
resource "jamfplatform_pro_package" "jcds_url" {
  display_name        = "MyApp 1.0.0 (URL)"
  file_name           = "MyApp-1.0.0.pkg"
  package_file_source = "https://artifacts.example.com/packages/MyApp-1.0.0.pkg"
}

# JCDS mode with a manifest sub-resource. `manifest_file_source` accepts
# either a local path or a URL. Idempotency on the manifest body uses
# direct equality with the server-stored `manifest` string, so changing
# only the on-disk filename without changing the body does NOT re-upload.
resource "jamfplatform_pro_package" "jcds_with_manifest" {
  display_name         = "MyApp 1.0.0 (with manifest)"
  file_name            = "MyApp-1.0.0.pkg"
  package_file_source  = "/path/to/MyApp-1.0.0.pkg"
  manifest_file_source = "/path/to/MyApp-1.0.0.plist"
}

# File-share DP with user-supplied hashes (FSDP-with-hashes mode). The
# binary lives on a customer-managed share; hashes are computed off
# Terraform's workstation and declared here. The server stores whatever
# the user PUTs without validation.
resource "jamfplatform_pro_package" "fsdp_with_hashes" {
  display_name = "Legacy installer"
  file_name    = "LegacyInstaller.pkg"
  category_id  = "5"
  info         = "Lives on the corp file-share DP"

  hash_type  = "MD5"
  hash_value = "d41d8cd98f00b204e9800998ecf8427e"

  md5    = "d41d8cd98f00b204e9800998ecf8427e"
  sha256 = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
  # sha3_512 omitted: only declare the hashes you have.
}

# Pure metadata-only (FSDP). No upload, no hash compute, no verification
# poll. Useful when the file lives on a customer-managed share and hash
# management happens entirely outside Terraform's view.
resource "jamfplatform_pro_package" "meta_only" {
  display_name                 = "Customer-managed installer"
  file_name                    = "CustomerInstaller.pkg"
  notes                        = "Bytes provisioned out-of-band by the IT team."
  reboot_required              = true
  os_requirements              = "14.0.x, 14.1.x"
  available_in_software_update = false
}
