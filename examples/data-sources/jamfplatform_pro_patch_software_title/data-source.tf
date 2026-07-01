# Look up a patch software title by ID.
data "jamfplatform_pro_patch_software_title" "example" {
  id = "6"
}

# Or by exact display name. Exactly one of id or name may be supplied.
data "jamfplatform_pro_patch_software_title" "by_name" {
  name = "8x8 Work"
}

output "patch_software_title_available_versions" {
  value = data.jamfplatform_pro_patch_software_title.example.available_versions
}

output "patch_software_title_version_packages" {
  value = data.jamfplatform_pro_patch_software_title.example.version_packages
}
