---
page_title: "AI Governance policies"
description: |-
  Manage the settings Jamf delivers to Claude Code, Claude Desktop and OpenAI Codex, and deploy them with blueprints.
---

# AI Governance policies

An **AI policy** is the managed configuration for one AI tool running on your Macs. You author it here; a blueprint delivers it.

The [AI Governance Configuration Guide](https://learn.jamf.com/r/en-US/ai-governance-configuration-guide/AI_Governance) covers the capability itself. Read it alongside this guide, which covers the Terraform half only. [Further reading](#further-reading) lists the pages to keep open while you work.

AI Governance has two halves. [**AI Visibility**](https://learn.jamf.com/r/en-US/ai-governance-configuration-guide/AI_Visibility) is a dashboard in Jamf Account, fed by the Jamf Protect agent. It reports which AI tools run on your fleet, what launches them, and which commands they run, including risky ones. It is read-only, the provider exposes none of it, and it is where you work out what a policy should say. [**AI Policies**](https://learn.jamf.com/r/en-US/ai-governance-configuration-guide/AI_Policies) is the half Terraform manages: the configuration, and the blueprint that delivers it. Everything below is that half.

## End to end

Write the policy, target a device group, deliver the published version.

```hcl
# 1. The policy. Applying it publishes version 1. schema_version is pinned to a
#    literal, so the schema the settings are written against changes only when you
#    change it. See "Pinning a schema version, or tracking the current one" below.
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
#    instead of writing a number keeps the blueprint moving with the policy. Jamf
#    refuses a blueprint naming a version that does not exist.
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

Done by hand, that last resource is the procedure in [Deploying a Policy with Blueprints](https://learn.jamf.com/r/en-US/ai-governance-configuration-guide/Deploying_a_Policy_with_Blueprints): drag the AI Governance component into a blueprint, choose a policy and a version, scope it, deploy. The Components library organises those components by AI product and hides the policy choice inside one. The provider's block works from IDs instead. `policies` is a list, each entry a `policy_id` and a `version`, and the order you write is the order the platform keeps.

### Pinning a schema version, or tracking the current one

Pin a literal, as the example above does. It is the safer default: the schema a policy is written against then moves only when you move it.

```hcl
resource "jamfplatform_ai_governance_policy" "pinned" {
  # ...
  schema_version = "2026-08-14"
}
```

Reading `current_schema_version` from the tool data source instead keeps the policy on whatever the platform publishes:

```hcl
data "jamfplatform_ai_governance_tool" "claude_code" {
  id = "com.anthropic.claudecode"
}

resource "jamfplatform_ai_governance_policy" "tracks_current" {
  # ...
  schema_version = data.jamfplatform_ai_governance_tool.claude_code.current_schema_version
}
```

That form has two costs, and the second is easy to miss. A new schema version becomes a change in your next plan, and the settings you wrote may need reconciling with it. And **the apply that moves the version on its own publishes nothing**: the platform compares the settings, not the schema version. A version change with unchanged settings mints no version, and blueprints keep delivering the one published against the older schema. "Drafts, versions and what actually reaches a device" below says what to do about it.

Either way, `terraform plan` warns when the version in use is no longer the current one, and `schema_drift` reports it in state. What the data source offers:

| Attribute | What it holds |
|---|---|
| `current_schema_version` | The version the platform publishes for the tool right now. |
| `schema_versions` | Every version the tool still accepts, newest first. |
| `schema_version` | Which version `settings_schema_json` describes. Defaults to the current one; set it to read an older one. |
| `settings_schema_json` | The JSON Schema for that version: what the settings may contain. |

## Drafts, versions and what actually reaches a device

A policy carries a **draft** and a history of **published versions**. Applying a change saves the draft and publishes it, so `published_version` moves forward. That is what `publish = true`, the default, means.

Nothing reaches a device until a blueprint delivers it. A blueprint pins a **version number**; it does not track the policy. That is why the end-to-end example above interpolates `published_version` into the blueprint's AI Governance component. Until a blueprint names the policy and is deployed, the policy is configuration nobody has received.

Set `publish = false` to stage changes without creating a version. Useful when someone else reviews and publishes in the Jamf Account admin UI. `has_draft` then reports that unpublished changes are waiting. The admin UI calls that state **Unpublished changes**, and its **Publish changes** action clears it. See [Publishing Changes to a Policy](https://learn.jamf.com/r/en-US/ai-governance-configuration-guide/Publishing_changes_to_an_AI_Governance_policy).

Publishing is skipped automatically when nothing changed: renaming a policy does not mint a version, because the platform compares the settings it holds against the ones sent.

That comparison is settings-only, and one consequence follows. **Moving `schema_version` forward without also changing the settings publishes nothing.** The policy's own `schema_drift` clears, but the version blueprints deliver is still the one published against the older schema. Change a setting in the same apply, or accept that the deployed version stays put until the next real change.


### When a publish fails

Saving the draft and publishing it are two calls, and the second can fail on its own: a transient error, or an integration granted `ai-policies:write` but not the publish route. When it does, the apply reports an error and Terraform still records the policy, including `has_draft = true`. A policy that exists is never left out of state, because policy names are not unique and a second create would leave two of them.

Blueprints keep delivering the previously published version until the publish succeeds. **The next apply retries it.** While `publish` is enabled, the surviving draft makes `has_draft` and `published_version` show as *known after apply* even when nothing else changed, so `terraform apply` has something to do and publishes the draft as it stands. Publishing it in the Jamf Account admin UI instead works just as well. The next refresh then reports `has_draft = false` and the plan goes quiet.

The same mechanism publishes a draft somebody saved in the admin UI on a policy managed with `publish = true`. That is what `publish = true` means. If the draft's settings differ from the configuration, the ordinary `settings_json` diff reverts them first, so what gets published is what Terraform holds. Set `publish = false` on policies whose publishing someone else owns.

### Destroying a policy a blueprint still references

Nothing stops you deleting a policy a deployed blueprint references. There is no refusal, no warning and no cleanup. The blueprint is left pointing at a version the platform will no longer serve, and the next change to that blueprint is rejected because the policy is archived.

Terraform cannot see this coming: nothing in the API reports which blueprints reference a policy. **Re-point or remove the blueprint's AI Governance component before destroying the policy.**

The [Deleting a Policy](https://learn.jamf.com/r/en-US/ai-governance-configuration-guide/Deleting_a_Policy) page opens with the same instruction. It is a procedure you follow, not a rule the platform enforces. Nothing stops you skipping it, in the admin UI or in Terraform.

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

That last form is the shortest route from an existing policy to a managed one. In Jamf Account, on the policy's row, choose **Download latest config** and you have exactly this document. It travels the other way too: the **Upload config** button in the admin UI's create-policy wizard takes the same JSON. A policy prototyped in the builder and one written in HCL are interchangeable.

One thing the wizard does that Terraform does not: it asks for an authentication method and its provider settings before showing any settings category. There is no wizard here, so those are keys in `settings_json` like every other key, and the tool's category reference below says which ones they are.

Formatting and key order are not significant. The value is compared as JSON, so reindenting or reordering keys produces no change.

### Where each tool's settings are documented

The category references group each tool's settings the way the admin UI's policy builder does. For Claude Code that means *Models & Reasoning*, *Identity & Compliance*, *Permissions*, *Sandbox Isolation*, *Managed MCP Server Policy*, *Hook Execution Policy* and the rest, which is the quickest way to find the handful of keys behind an outcome you have in mind. The vendor documentation is then authoritative on what each key actually does.

| Product | Category reference | Vendor documentation |
|---|---|---|
| Claude Code | [Claude Code Configuration Categories Reference](https://learn.jamf.com/r/en-US/ai-governance-configuration-guide/Claude_Code_Configuration_Categories_Reference) | [code.claude.com/docs/en/settings](https://code.claude.com/docs/en/settings); schema also published at [json.schemastore.org/claude-code-settings.json](https://json.schemastore.org/claude-code-settings.json) |
| Claude Desktop | [Claude Desktop Configuration Categories Reference](https://learn.jamf.com/r/en-US/ai-governance-configuration-guide/Claude_Desktop_Configuration_Categories_Reference) | [support.claude.com/en/articles/12622667](https://support.claude.com/en/articles/12622667) |
| OpenAI Codex | [OpenAI Codex Configuration Categories Reference](https://learn.jamf.com/r/en-US/ai-governance-configuration-guide/OpenAI_Codex_Configuration_Categories_Reference) | [developers.openai.com/codex/config-reference](https://developers.openai.com/codex/config-reference) |

The schema each policy is actually validated against is the one the platform serves, which you can read directly:

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

- **Error.** A setting of the wrong type, outside its accepted values, or missing when required. The platform refuses these writes, so failing the plan beats failing the apply.
- **Warning.** A setting the schema does not declare, where the tool accepts undeclared settings. The platform stores it and the tool never applies it, so a typo here is silently inert. This is the one problem the platform itself will never report.

A few schema rules are not checked: conditional (`if`/`then`) rules, and format assertions. The platform still enforces everything, so an apply can fail where a plan passed. The diagnostic says as much when it happens.

#### A policy authored in the Jamf admin UI may hold settings its schema does not declare

The Jamf Account admin UI can write settings that the schema published for that version does not list. Importing such a policy, or copying its settings into a configuration, therefore raises the undeclared-setting **warning** above on a policy that is working correctly.

The warning is advisory here: the platform stores the setting and reports it back unchanged. The known instance is `banner` on Claude Desktop. The `com.anthropic.claudefordesktop` schema for `2026-06-03` declares seven settings and not `banner`, yet a policy created in the admin UI carries it, and writing the same setting from Terraform is accepted. Leave it in place.

## Schema versions and drift

A tool publishes new settings schema versions over time. A policy written against an older one keeps working, and reports `schema_drift`:

```hcl
data "jamfplatform_ai_governance_policies" "needs_review" {
  schema_drift_only = true
}
```

`terraform plan` also warns on a drifted policy, naming the current version. Moving forward means setting `schema_version` to it and reconciling `settings_json`. Settings the older schema did not declare become available, and any it declared that the newer one dropped must go.

A `tool_id` or `schema_version` the catalogue no longer lists fails the plan when the configuration has just changed it. That is a typo, caught before an apply. It only warns when the value is unchanged from the last apply: a version withdrawn from the catalogue must not fail plans whose real changes are elsewhere, and the write itself reports the problem if the platform has stopped accepting it.

Pinning `schema_version` to a literal is the conservative choice, for the reasons set out in "Pinning a schema version, or tracking the current one" above.

## Requirements

### The capability

AI Governance is enabled once, in Jamf Account, and Terraform cannot do it for you. The [Requirements](https://learn.jamf.com/r/en-US/ai-governance-configuration-guide/Requiremements) page is authoritative. In outline:

- One of the cloud platform offerings that includes AI Governance: Jamf for Mac, Jamf for Mac – Higher Education, Business Plan or Enterprise Plan. Standalone product subscriptions are not eligible, and no configuration change makes them so.
- A **platform environment** in Jamf Account grouping your Jamf Pro and Jamf Protect tenants, all in one region. Jamf Security Cloud is not part of this grouping.
- **OIDC SSO through Jamf Account** enabled in Jamf Pro. The policy builder is reached through Jamf Account.
- Jamf Protect deployed to the environment. This is what AI Visibility runs on, and the capability will not enable without it.
- A Jamf Account role, scoped to that environment, carrying the AI Governance privileges: **Policies** (create, view, edit, delete) and **Visibility** (view). Plus, in Jamf Pro, at least **Blueprints** create/read/update/delete.

That environment is the same platform environment `environment_id` names below. It is why these constructs are environment-scoped, not tenant-scoped.

### The provider's API integration

AI Governance is reached under an **environment-scoped** API integration. Set `environment_id` (or `JAMFPLATFORM_ENVIRONMENT_ID`) on the provider. A tenant-scoped integration is not supported for these constructs.

The integration needs the **AI policies** permission, under **Compliance** in Jamf Account's permission picker. API capability `ai-policies`. That capability is the whole surface: every action it exposes is one of create, read, update or delete, and each resource and data source page lists which of the four it uses.

An integration's permission is granted separately from a person's role: a user clicking through the policy builder and a Terraform run authenticating with client credentials are authorised independently, so granting one does not grant the other.

## Further reading

| Topic | Page |
|---|---|
| The capability end to end | [AI Governance](https://learn.jamf.com/r/en-US/ai-governance-configuration-guide/AI_Governance) |
| Plans, privileges and prerequisites | [Requirements](https://learn.jamf.com/r/en-US/ai-governance-configuration-guide/Requiremements) |
| What is running on the fleet, and what to configure because of it | [AI Visibility](https://learn.jamf.com/r/en-US/ai-governance-configuration-guide/AI_Visibility), [Viewing the AI Visibility Dashboard](https://learn.jamf.com/r/en-US/ai-governance-configuration-guide/Viewing_the_AI_Visibility_Dashboard) |
| Enabling the capability | [Enabling AI Visibility](https://learn.jamf.com/r/en-US/ai-governance-configuration-guide/Enabling_AI_Visibility) |
| Authoring a policy in the admin UI, and the JSON upload | [Creating a New Policy](https://learn.jamf.com/r/en-US/ai-governance-configuration-guide/Creating_a_New_Policy) |
| Publishing, and the *Unpublished changes* state | [Publishing Changes to a Policy](https://learn.jamf.com/r/en-US/ai-governance-configuration-guide/Publishing_changes_to_an_AI_Governance_policy) |
| Delivering a policy with a blueprint | [Deploying a Policy with Blueprints](https://learn.jamf.com/r/en-US/ai-governance-configuration-guide/Deploying_a_Policy_with_Blueprints) |
| Removing a policy safely | [Deleting a Policy](https://learn.jamf.com/r/en-US/ai-governance-configuration-guide/Deleting_a_Policy) |
| What each tool's settings mean | the per-product references in [Where each tool's settings are documented](#where-each-tools-settings-are-documented) above |
