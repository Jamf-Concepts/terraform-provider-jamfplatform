# Look up a patch software title by ID. (Name lookup is not supported — the
# classic list endpoint exposes no display name through the SDK.)
data "jamfplatform_pro_patch_software_title" "example" {
  id = "6"
}

output "patch_software_title_available_versions" {
  value = data.jamfplatform_pro_patch_software_title.example.available_versions
}

output "patch_software_title_version_packages" {
  value = data.jamfplatform_pro_patch_software_title.example.version_packages
}
