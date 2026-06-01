# List every App Installer deployment (expanded form with resolved app/site/
# category/smart-group references and per-deployment computer status counts).
data "jamfplatform_pro_app_installers" "all" {}

# Narrow the list to deployments whose name contains a substring.
data "jamfplatform_pro_app_installers" "editor" {
  name_substring = "Editor"
}

output "editor_deployment_ids" {
  value = [for d in data.jamfplatform_pro_app_installers.editor.deployments : d.id]
}
