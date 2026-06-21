# Look up a single App Installer catalog title by ID.
data "jamfplatform_pro_app_installer_title" "editor" {
  id = "518" # Jamf Composer
}

output "editor_version" {
  value = data.jamfplatform_pro_app_installer_title.editor.version
}

# Use the title's name to drive an App Installer (the resource resolves the name
# to an ID).
resource "jamfplatform_pro_app_installer" "editor" {
  name            = "Jamf Composer"
  app_title_name  = data.jamfplatform_pro_app_installer_title.editor.title_name
  deployment_type = "SELF_SERVICE"
  update_behavior = "AUTOMATIC"
}
