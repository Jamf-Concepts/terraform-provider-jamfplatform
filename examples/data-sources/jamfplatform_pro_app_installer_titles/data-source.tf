# Read the whole App Installer catalog.
data "jamfplatform_pro_app_installer_titles" "all" {}

# Narrow the catalog by a case-insensitive name substring.
data "jamfplatform_pro_app_installer_titles" "editors" {
  name_substring = "Editor"
}

# Resolve a single title's name to pass to an App Installer's app_title_name.
output "first_editor_title_name" {
  value = data.jamfplatform_pro_app_installer_titles.editors.titles[0].title_name
}
