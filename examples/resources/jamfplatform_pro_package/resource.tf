# Cloud distribution point upload from a local path. The provider uploads
# the file to the Jamf Cloud Distribution Point, then waits until Jamf Pro
# has calculated every hash and `cloud_transfer_status` becomes "READY".
# The hash attributes are read-only in this mode — setting any of them in
# configuration is rejected before the change runs.
resource "jamfplatform_pro_package" "cloud_dp_local" {
  display_name        = "MyApp 1.0.0"
  file_name           = "MyApp-1.0.0.pkg"
  category_id         = "-1"
  info                = "Internal build of MyApp"
  reboot_required     = false
  priority            = 10
  package_file_source = "/path/to/MyApp-1.0.0.pkg"

  # Optional integrity check: verified against the file locally before
  # anything is uploaded. A mismatch fails the apply and skips the upload —
  # useful for catching on-disk corruption.
  package_file_source_checksum = "0123456789abcdef..." # hex sha3-512

  # Uploads can take minutes for very large files. The upload inherits this
  # deadline.
  timeouts {
    create = "30m"
    update = "30m"
  }
}

# Cloud distribution point upload from an HTTPS URL. The provider downloads
# the URL to a temporary file (8 GiB limit, up to 10 redirects), then uploads
# it the same way as the local-path example.
resource "jamfplatform_pro_package" "cloud_dp_url" {
  display_name        = "MyApp 1.0.0 (URL)"
  file_name           = "MyApp-1.0.0.pkg"
  package_file_source = "https://artifacts.example.com/packages/MyApp-1.0.0.pkg"
}

# Upload with a manifest. `manifest_file_source` accepts either a local path
# or a URL. The manifest is re-uploaded only when its contents change, so
# changing only the on-disk filename without changing the body does not
# re-upload.
resource "jamfplatform_pro_package" "cloud_dp_with_manifest" {
  display_name         = "MyApp 1.0.0 (with manifest)"
  file_name            = "MyApp-1.0.0.pkg"
  package_file_source  = "/path/to/MyApp-1.0.0.pkg"
  manifest_file_source = "/path/to/MyApp-1.0.0.plist"
}

# Distribution point with supplied hashes. The file lives on a distribution
# point you manage; the hashes are calculated elsewhere and declared here.
# Jamf Pro stores the values as given, without validating them.
resource "jamfplatform_pro_package" "supplied_hashes" {
  display_name = "Legacy installer"
  file_name    = "LegacyInstaller.pkg"
  category_id  = "5"
  info         = "Lives on a self-managed distribution point"

  hash_type  = "MD5"
  hash_value = "d41d8cd98f00b204e9800998ecf8427e"

  md5    = "d41d8cd98f00b204e9800998ecf8427e"
  sha256 = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
  # sha3_512 omitted: only declare the hashes you have.
}

# Metadata only. No file is uploaded. Useful when the file lives on a
# self-managed distribution point and hash management happens entirely
# outside Terraform.
resource "jamfplatform_pro_package" "meta_only" {
  display_name                 = "Customer-managed installer"
  file_name                    = "CustomerInstaller.pkg"
  notes                        = "Bytes provisioned out-of-band by the IT team."
  reboot_required              = true
  os_requirements              = "14.0.x, 14.1.x"
  available_in_software_update = false
}
