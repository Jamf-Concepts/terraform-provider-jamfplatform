# Blueprints are authored with `component_blocks`, an ordered list where each block appears as a
# step in the Jamf Blueprints editor, with its own name, its own optional activation condition, and
# its own components. Blocks are applied in the order listed, and a block may contain more than one
# component.

# Single block: Software Update Settings
resource "jamfplatform_blueprints_blueprint" "software_update_settings" {
  name        = "Software Update Settings"
  description = "Managed by Terraform"
  deployed    = true

  device_groups = ["fce3d9a5-8660-42ff-a95e-625e7b53b48a"]

  component_blocks = [
    {
      name = "Software Update Settings"
      software_update_settings = {
        allow_standard_user_os_updates           = true
        automatic_download                       = "AlwaysOn"
        automatic_install_os_updates             = "AlwaysOn"
        automatic_install_security_updates       = "AlwaysOn"
        beta_program_enrollment                  = "Allowed"
        deferral_combined_period_days            = 7
        deferral_major_period_days               = 30
        deferral_minor_period_days               = 14
        deferral_system_period_days              = 3
        notifications_enabled                    = true
        rapid_security_response_enabled          = true
        rapid_security_response_rollback_enabled = false
        recommended_cadence                      = "Newest"

        beta_offer_programs = [{
          token       = "beta-token-1"
          description = "iOS 18 Beta Program"
          }, {
          token       = "beta-token-2"
          description = "macOS Sequoia Beta Program"
        }]
      }
    },
  ]
}

# Two components in a single block. A block may carry more than one component.
resource "jamfplatform_blueprints_blueprint" "baseline_restrictions" {
  name        = "Baseline Restrictions"
  description = "Managed by Terraform"
  deployed    = true

  device_groups = ["fce3d9a5-8660-42ff-a95e-625e7b53b48a"]

  component_blocks = [
    {
      name = "Baseline Restrictions"
      passcode_policy = {
        require_passcode = true
        minimum_length   = 6
      }
      math_settings = {
        calculator_scientific_mode_enabled   = true
        system_behavior_keyboard_suggestions = true
        system_behavior_math_notes           = true
      }
    },
  ]
}

# Multiple blocks. Each is a separate step, applied in order. The same component type may appear in
# more than one block, and each block can carry its own activation condition.
resource "jamfplatform_blueprints_blueprint" "multi_step" {
  name        = "Multi-Step Components"
  description = "Managed by Terraform"
  deployed    = false

  device_groups = ["fce3d9a5-8660-42ff-a95e-625e7b53b48a"]

  component_blocks = [
    {
      name = "Passcode Policy"
      passcode_policy = {
        require_passcode = true
        minimum_length   = 6
      }
    },
    {
      name = "Latest OS Software Updates"
      software_update = {
        ignore_major_versions = true
        deployment_time       = "02:00"
        enforce_after_days    = 7
      }
    },
    {
      name = "Math Settings"
      math_settings = {
        calculator_scientific_mode_enabled   = true
        system_behavior_keyboard_suggestions = true
        system_behavior_math_notes           = true
      }
    },
  ]
}

# Legacy configuration profile payloads inside a block.
resource "jamfplatform_blueprints_blueprint" "legacy_payloads_example" {
  name        = "Restrictions for Safari"
  description = "Managed by Terraform"
  deployed    = true

  device_groups = ["fce3d9a5-8660-42ff-a95e-625e7b53b48a"]

  component_blocks = [
    {
      name = "Safari Restrictions"
      legacy_payloads = [
        {
          payload_type = "com.apple.applicationaccess"
          settings = jsonencode({
            allowSafariHistoryClearing = false
            allowSafariPrivateBrowsing = false
          })
        }
      ]
    },
  ]
}

# Custom declarations inside a block.
resource "jamfplatform_blueprints_blueprint" "customdeclaration" {
  name        = "Custom Declarations"
  description = "Managed by Terraform"
  deployed    = true

  device_groups = ["fce3d9a5-8660-42ff-a95e-625e7b53b48a"]

  component_blocks = [
    {
      name = "Custom Declarations"
      custom_declarations = {
        declaration = [{
          channel = "SYSTEM"
          kind    = "CONFIGURATION"
          type    = "com.apple.configuration.softwareupdate.settings"
          payload = jsonencode({
            Beta = {
              RequireProgram = {
                Token       = "<beta-token-here>",
                Description = "AppleSeed for IT"
              },
              ProgramEnrollment = "AlwaysOn"
            }
          })
          }, {
          channel = "USER"
          kind    = "ASSET"
          type    = "com.apple.asset.credential.userpassword"
          payload = jsonencode({
            Reference = {
              DataURL     = "https://somewhere.com/something.plist",
              ContentType = "application/plist"
            }
          })
        }]
      }
    },
  ]
}

# Per-block activation conditions.
#
# An activation condition further restricts which scoped devices a block applies to. Author it in
# the Jamf UI ("Activation conditions" editor -> Text view) and copy it here verbatim. See the
# syntax reference:
# https://learn.jamf.com/r/en-US/jamf-pro-blueprints-configuration-guide/Activation_Condition_Expression_Reference
#
# Device groups in the expression are referenced by their Platform UUID, so a managed device group
# can be referenced by its `id` with ordinary Terraform interpolation, keeping the condition in
# sync with the group it points at.
resource "jamfplatform_device_group" "shared_ipads" {
  name        = "Shared iPads"
  group_type  = "smart"
  device_type = "mobile"
  description = "Managed by Terraform"
  criteria = [
    {
      criteria = "Model"
      operator = "like"
      value    = "iPad"
    },
  ]
}

resource "jamfplatform_blueprints_blueprint" "activation_conditions_example" {
  name        = "Shared iPad Software Updates"
  description = "Managed by Terraform"
  deployed    = true

  device_groups = [jamfplatform_device_group.shared_ipads.id]

  component_blocks = [
    {
      name = "Shared iPad Software Updates"
      # Only activate on supervised iPads that belong to the managed device group above.
      # The group ID is interpolated from the resource, so the condition tracks the group.
      activation_conditions = "ANY @property(jamf.device.groups) IN {'${jamfplatform_device_group.shared_ipads.id}'} AND @status(device.model.family) == 'iPad'"
      software_update = {
        ignore_major_versions = true
        deployment_time       = "02:00"
        enforce_after_days    = 7
      }
    },
  ]
}

# Deliver a managed AI tool configuration. The blueprint pins a published policy
# version, so interpolating `published_version` keeps the two moving together.
# Jamf refuses a blueprint that names a version which does not exist.
data "jamfplatform_ai_governance_tool" "claude_code" {
  id = "com.anthropic.claudecode"
}

resource "jamfplatform_ai_governance_policy" "engineering" {
  name           = "Claude Code — Engineering"
  tool_id        = data.jamfplatform_ai_governance_tool.claude_code.id
  schema_version = data.jamfplatform_ai_governance_tool.claude_code.current_schema_version

  settings_json = jsonencode({
    model                  = "sonnet"
    availableModels        = ["sonnet", "haiku"]
    enforceAvailableModels = true
  })
}

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

resource "jamfplatform_blueprints_blueprint" "ai_governance" {
  name          = "AI Governance — Engineering"
  description   = "Managed Claude Code settings."
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
