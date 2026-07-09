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
