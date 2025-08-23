data "jamfplatform_blueprints_components" "all" {}

output "all_components" {
  value = data.jamfplatform_blueprints_components.all
}
