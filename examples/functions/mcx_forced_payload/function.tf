# Build a macOS managed (forced) preferences payload and deliver it via a
# configuration profile. Whole numbers become integers, fractional numbers
# reals, booleans/strings map directly, lists become arrays, nested objects
# become dictionaries.
resource "jamfplatform_pro_macos_configuration_profile" "app_settings" {
  general = {
    name = "Example App Configuration"
    payloads = provider::jamfplatform::mcx_forced_payload("com.example.app", {
      RotateWithinHours = 24
      AdminBase         = "https://admin.example.com"
      Browsers          = ["edge", "chrome", "firefox"]
    })
  }

  scope = {
    targets = {
      all_computers = true
    }
  }
}

# A real-world example: Visual Studio Code's enterprise policies
# (https://code.visualstudio.com/docs/setup/enterprise). Vendors commonly
# document these as a payload whose PayloadType is the app's own preference
# domain with the policy keys inline. That shape is a valid profile and will
# deploy through Jamf, but Jamf Pro's admin UI cannot render it — the settings
# are invisible to admins. Wrapping the same keys in the MCX "Forced" envelope
# (which is exactly what this function emits) deploys the same preferences AND
# shows them in the Jamf Pro UI under Application & Custom Settings.
resource "jamfplatform_pro_macos_configuration_profile" "vscode_policies" {
  general = {
    name = "VS Code - Enterprise Policies"
    payloads = provider::jamfplatform::mcx_forced_payload("com.microsoft.VSCode", {
      UpdateMode        = "none"
      TelemetryLevel    = "off"
      EnableFeedback    = false
      ChatAgentMode     = false
      ChatMCP           = "none"
      AllowedExtensions = jsonencode(["ms-python.python", "golang.go"])
    })
  }

  scope = {
    targets = {
      all_computers = true
    }
  }
}
