data "jamfplatform_ai_governance_tools" "all" {}

output "governable_tools" {
  description = "Every AI tool Jamf can govern, with the settings schema version each one currently publishes."
  value = {
    for tool in data.jamfplatform_ai_governance_tools.all.tools :
    tool.id => tool.current_schema_version
  }
}
