---
page_title: "AI Governance policies"
description: |-
  Manage the settings Jamf delivers to Claude Code, Claude Desktop and OpenAI Codex, and deploy them with blueprints.
---

# AI Governance policies

An **AI policy** is the managed configuration for one AI tool running on your Macs. You author it here; a blueprint delivers it.

## End to end

Three resources: write the policy, target a device group, deliver the published version.

```hcl
# 1. The policy. Applying it publishes version 1. schema_version is pinned to a
#    literal, so the schema the settings are written against changes only when you
#    change it — see "Pinning a schema version, or tracking the current one" below.
resource "jamfplatform_ai_governance_policy" "engineering" {
  name           = "Claude Code — Engineering"
  description    = "Managed Claude Code settings for the engineering fleet."
  tool_id        = "com.anthropic.claudecode"
  schema_version = "2026-08-14"

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

# 2. Who receives it.
resource "jamfplatform_device_group" "engineering_macs" {
  name        = "Engineering Macs"
  group_type  = "smart"
  device_type = "computer"

  criteria = [{
    criteria = "Department"
    operator = "is"
    value    = "Engineering"
  }]
}

# 3. The blueprint delivers a published version. Interpolating published_version
#    rather than writing a number keeps the blueprint moving with the policy —
#    Jamf refuses a blueprint naming a version that does not exist.
resource "jamfplatform_blueprints_blueprint" "ai_governance" {
  name          = "AI Governance — Engineering"
  deployed      = true
  device_groups = [jamfplatform_device_group.engineering_macs.id]

  component_blocks = [
    {
      name = "AI Governance"
      ai_governance = {
        policies = [
          {
            policy_id = jamfplatform_ai_governance_policy.engineering.id
            version   = jamfplatform_ai_governance_policy.engineering.published_version
          },
        ]
      }
    },
  ]
}
```

Change the policy's settings and the next apply publishes version 2, the blueprint's pinned version follows, and the blueprint redeploys. Nothing else to do.

### Pinning a schema version, or tracking the current one

Pinning a literal, as the example above does, is the safer default — the schema a policy is written against then moves only when you move it:

```hcl
resource "jamfplatform_ai_governance_policy" "pinned" {
  # ...
  schema_version = "2026-08-14"
}
```

Reading `current_schema_version` from the tool data source instead keeps the policy on whatever Jamf publishes:

```hcl
data "jamfplatform_ai_governance_tool" "claude_code" {
  id = "com.anthropic.claudecode"
}

resource "jamfplatform_ai_governance_policy" "tracks_current" {
  # ...
  schema_version = data.jamfplatform_ai_governance_tool.claude_code.current_schema_version
}
```

That form has two costs, and the second is easy to miss. A new schema version published by Jamf becomes a change in your next plan, and the settings you wrote may need reconciling with it. And **the apply that moves the version on its own publishes nothing**: Jamf compares the settings, not the schema version, so a version change with unchanged settings mints no version and blueprints keep delivering the one published against the older schema. "Drafts, versions and what actually reaches a device" below sets out what to do about it.

Either way, `terraform plan` warns when the version in use is no longer the current one, and `schema_drift` reports it in state. What the data source offers:

| Attribute | What it holds |
|---|---|
| `current_schema_version` | The version Jamf publishes for the tool right now. |
| `schema_versions` | Every version the tool still accepts, newest first. |
| `schema_version` | Which version `settings_schema_json` describes — defaults to the current one, or set it to read an older one. |
| `settings_schema_json` | The JSON Schema for that version: what the settings may contain. |

## Drafts, versions and what actually reaches a device

A policy carries a **draft** and a history of **published versions**. Applying a change saves the draft and publishes it, so `published_version` moves forward — that is what `publish = true`, the default, means.

Nothing reaches a device until a blueprint delivers it, and a blueprint pins a **version number** rather than tracking the policy — which is why the end-to-end example above interpolates `published_version` into the blueprint's AI Governance component. Until a blueprint names the policy and is deployed, the policy is configuration nobody has received.

Set `publish = false` to stage changes without creating a version — useful when someone else reviews and publishes in the Jamf Account admin UI. `has_draft` then reports that unpublished changes are waiting.

Publishing is skipped automatically when nothing changed: renaming a policy does not mint a version, because Jamf compares the settings it holds against the ones sent.

That comparison is settings-only, and it has one consequence worth knowing. **Moving `schema_version` forward without also changing the settings publishes nothing.** The policy's own `schema_drift` clears, but the version blueprints deliver is still the one published against the older schema. Change a setting in the same apply — or accept that the deployed version stays where it is until the next real change.

### Destroying a policy a blueprint still references

Jamf lets you delete a policy a deployed blueprint references. It does not refuse, warn, or clean up the blueprint — the blueprint is left pointing at a version Jamf will no longer serve, and the next change to that blueprint is rejected because the policy is archived.

Terraform cannot see this coming: nothing in the API reports which blueprints reference a policy. **Re-point or remove the blueprint's AI Governance component before destroying the policy.**

## Writing `settings_json`

The settings are the tool vendor's own configuration format, not Jamf's. Each tool publishes a JSON Schema per `schema_version`, and that schema is the authority on what the settings may contain.

Author them however suits the size of the policy:

```hcl
# inline
settings_json = jsonencode({ model = "sonnet" })

# from a file
settings_json = file("${path.module}/claude-code.json")

# from a configuration exported out of the Jamf Account admin UI
settings_json = file("${path.module}/exported-policy.json")
```

Formatting and key order are not significant — the value is compared as JSON, so reindenting or reordering keys produces no change.

### Where each tool's settings are documented

| Product | Jamf's category reference | Vendor documentation |
|---|---|---|
| Claude Code | [Claude Code Configuration Categories Reference](https://learn.jamf.com/r/en-US/ai-governance-configuration-guide/Claude_Code_Configuration_Categories_Reference) | [code.claude.com/docs/en/settings](https://code.claude.com/docs/en/settings) — schema also published at [json.schemastore.org/claude-code-settings.json](https://json.schemastore.org/claude-code-settings.json) |
| Claude Desktop | [Claude Desktop Configuration Categories Reference](https://learn.jamf.com/r/en-US/ai-governance-configuration-guide/Claude_Desktop_Configuration_Categories_Reference) | [support.claude.com/en/articles/12622667](https://support.claude.com/en/articles/12622667) |
| OpenAI Codex | [OpenAI Codex Configuration Categories Reference](https://learn.jamf.com/r/en-US/ai-governance-configuration-guide/OpenAI_Codex_Configuration_Categories_Reference) | [developers.openai.com/codex/config-reference](https://developers.openai.com/codex/config-reference) |

The schema each policy is actually validated against is the one Jamf serves, which you can read directly:

```hcl
data "jamfplatform_ai_governance_tool" "claude_code" {
  id = "com.anthropic.claudecode"
}

output "declared_settings" {
  value = keys(jsondecode(data.jamfplatform_ai_governance_tool.claude_code.settings_schema_json).properties)
}
```

### What is checked during `terraform plan`

The provider validates `settings_json` against that schema before an apply, and the difference between an error and a warning is the schema's, not a preference:

- **Error** — a setting of the wrong type, outside its accepted values, or missing when required. Jamf refuses these writes, so failing the plan is strictly better than failing the apply.
- **Warning** — a setting the schema does not declare, where the tool accepts undeclared settings. Jamf stores it and the tool never applies it, so a typo here is silently inert. This is the one problem Jamf itself will never report.

A few schema rules are not checked — conditional (`if`/`then`) rules, and format assertions. Jamf still enforces everything, so an apply can fail where a plan passed; the diagnostic says as much when it happens.

#### A policy authored in the Jamf admin UI may hold settings its schema does not declare

The Jamf Account admin UI can write settings that the schema published for that version does not list. Importing such a policy, or copying its settings into a configuration, therefore raises the undeclared-setting **warning** above on a policy that is working correctly.

The warning is advisory in this case: Jamf stores the setting and reports it back unchanged. The known instance is `banner` on Claude Desktop — the `com.anthropic.claudefordesktop` schema for `2026-06-03` declares seven settings and not `banner`, yet a policy created in the admin UI carries it, and writing the same setting from Terraform is accepted. Leave it in place.

## Schema versions and drift

A tool publishes new settings schema versions over time. A policy written against an older one keeps working, and reports `schema_drift`:

```hcl
data "jamfplatform_ai_governance_policies" "needs_review" {
  schema_drift_only = true
}
```

`terraform plan` also warns on a drifted policy, naming the current version. Moving forward means setting `schema_version` to it and reconciling `settings_json` — settings the older schema did not declare become available, and any it declared that the newer one dropped must go.

Pinning `schema_version` to a literal is the conservative choice, for the reasons set out in "Pinning a schema version, or tracking the current one" above.

## Requirements

AI Governance is reached under an **environment-scoped** API integration — set `environment_id` (or `JAMFPLATFORM_ENVIRONMENT_ID`) on the provider. A tenant-scoped integration is not supported for these constructs.

The integration needs the `ai-policies` privileges; each resource and data source page lists the ones it uses.
