# Look up an App Installer deployment by exact name.
data "jamfplatform_pro_app_installer" "by_name" {
  name = "010 Editor (Self Service)"
}

output "deployment_id" {
  value = data.jamfplatform_pro_app_installer.by_name.id
}

# Look up an App Installer deployment by ID.
data "jamfplatform_pro_app_installer" "by_id" {
  id = "177"
}
