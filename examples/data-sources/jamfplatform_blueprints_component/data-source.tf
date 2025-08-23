data "jamfplatform_blueprints_component" "example" {
  id = "com.jamf.ddm.safari-settings"
}

output "component_identifier" {
  value = data.jamfplatform_blueprints_component.example.identifier
}

output "component_name" {
  value = data.jamfplatform_blueprints_component.example.name
}

output "component_description" {
  value = data.jamfplatform_blueprints_component.example.description
}

output "supported_operating_systems" {
  value = data.jamfplatform_blueprints_component.example.meta.supported_os
}
