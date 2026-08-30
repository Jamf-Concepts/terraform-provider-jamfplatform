# Read the catalogue so the policy tracks whatever settings schema version Jamf
# currently publishes for the tool, instead of pinning one that will drift.
data "jamfplatform_ai_governance_tool" "claude_code" {
  id = "com.anthropic.claudecode"
}

resource "jamfplatform_ai_governance_policy" "claude_code_engineering" {
  name           = "Claude Code — Engineering"
  description    = "Managed Claude Code settings for the engineering fleet."
  tool_id        = data.jamfplatform_ai_governance_tool.claude_code.id
  schema_version = data.jamfplatform_ai_governance_tool.claude_code.current_schema_version

  settings_json = jsonencode({
    model                  = "sonnet"
    availableModels        = ["sonnet", "haiku"]
    enforceAvailableModels = true
    includeCoAuthoredBy    = false

    permissions = {
      allow = ["Bash(git *)", "Read"]
      deny  = ["Read(./.env)", "Read(./secrets/**)"]
    }
  })
}

# Larger policies read better from a file. The contents are compared as JSON, so
# formatting and key order do not produce a change.
resource "jamfplatform_ai_governance_policy" "claude_code_from_file" {
  name           = "Claude Code — Contractors"
  tool_id        = data.jamfplatform_ai_governance_tool.claude_code.id
  schema_version = data.jamfplatform_ai_governance_tool.claude_code.current_schema_version
  settings_json  = file("${path.module}/claude-code-contractors.json")
}

# Stage changes without publishing them, to review in the Jamf Account admin UI
# before a new version is created. `has_draft` reports that one is waiting.
resource "jamfplatform_ai_governance_policy" "codex_draft" {
  name           = "Codex — Pilot"
  tool_id        = "com.openai.codex"
  schema_version = "2026-06-30"
  publish        = false

  settings_json = jsonencode({
    config = {
      approval_policy = "on-request"
    }
  })
}

# Deploying a published version to devices is a separate step: a blueprint's AI
# Governance component references the policy and the version to deliver.
output "claude_code_engineering_deployable_version" {
  description = "The published version a blueprint's AI Governance component should reference."
  value       = jamfplatform_ai_governance_policy.claude_code_engineering.published_version
}
