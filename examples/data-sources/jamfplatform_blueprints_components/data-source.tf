data "jamfplatform_blueprints_components" "all" {
  provider = jamfplatform.blueprints
}

output "all_components" {
  value = data.jamfplatform_blueprints_components.all
}
