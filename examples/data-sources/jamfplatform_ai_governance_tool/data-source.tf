# Reads the tool's current settings schema version, and the schema document that
# says what its settings may contain.
data "jamfplatform_ai_governance_tool" "claude_code" {
  id = "com.anthropic.claudecode"
}

# Read an older schema version that existing policies are still written against.
data "jamfplatform_ai_governance_tool" "claude_code_previous" {
  id             = "com.anthropic.claudecode"
  schema_version = "2026-05-19"
}

output "claude_code_settings_keys" {
  description = "Every setting Claude Code's current schema declares."
  value       = keys(jsondecode(data.jamfplatform_ai_governance_tool.claude_code.settings_schema_json).properties)
}
