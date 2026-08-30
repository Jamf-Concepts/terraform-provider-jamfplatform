# Policy names are not unique, so a policy is read by ID.
data "jamfplatform_ai_governance_policy" "claude_code" {
  id = "ae9db2cc-8480-48ed-aa29-704b0f515980"
}

output "settings" {
  description = "The policy's current settings."
  value       = jsondecode(data.jamfplatform_ai_governance_policy.claude_code.settings_json)
}

output "deployable_version" {
  description = "The published version a blueprint's AI Governance component should reference."
  value       = data.jamfplatform_ai_governance_policy.claude_code.published_version
}
