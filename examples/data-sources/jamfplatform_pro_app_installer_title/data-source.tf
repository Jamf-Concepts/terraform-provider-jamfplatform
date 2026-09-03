# Look up a single App Installer catalog title by ID.
data "jamfplatform_pro_app_installer_title" "composer" {
  id = "Composer"
}

output "composer_version" {
  value = data.jamfplatform_pro_app_installer_title.composer.version
}

# Read a historical version of the same title. The package hash, minimum OS
# version and signing identity all move between versions.
data "jamfplatform_pro_app_installer_title" "composer_pinned" {
  id      = "Composer"
  version = "11.30.1"
}

output "composer_pinned_minimum_os" {
  value = data.jamfplatform_pro_app_installer_title.composer_pinned.minimum_os_version
}

# Use the title's name to drive an App Installer (the resource resolves the name
# to an ID).
resource "jamfplatform_pro_app_installer" "composer" {
  name            = "Jamf Composer"
  app_title_name  = data.jamfplatform_pro_app_installer_title.composer.title_name
  deployment_type = "SELF_SERVICE"
  update_behavior = "AUTOMATIC"
}
