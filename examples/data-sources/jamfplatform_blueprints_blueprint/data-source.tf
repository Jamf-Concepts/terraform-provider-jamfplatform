data "jamfplatform_blueprints_blueprint" "example" {
  # You can look up by either id or name
  # id   = "12345678-90ab-cdef-1234-567890abcdef"
  provider = jamfplatform.blueprints
  name     = "Blueprint Name"
}

output "blueprint_example_all" {
  value = data.jamfplatform_blueprints_blueprint.example
}
