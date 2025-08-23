data "jamfplatform_blueprints_blueprint" "example" {
  # You can look up by either id or name
  # id   = "12345678-90ab-cdef-1234-567890abcdef"
  provider = jamfplatform.blueprints
  name     = "Blueprint Name"
}

output "blueprint_id" {
  value = data.jamfplatform_blueprint.example.blueprint_id
}

output "blueprint_description" {
  value = data.jamfplatform_blueprint.example.description
}

output "blueprint_device_groups" {
  value = data.jamfplatform_blueprint.example.device_groups
}

output "blueprint_steps" {
  value = data.jamfplatform_blueprint.example.steps
}
