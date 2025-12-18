data "jamfplatform_blueprints" "all" {
}

data "jamfplatform_blueprints" "accessibility" {
  search = "Accessibility Blueprint"
}

output "all_blueprints" {
  value = data.jamfplatform_blueprints.all
}

output "accessibility_blueprints" {
  value = data.jamfplatform_blueprints.accessibility
}
