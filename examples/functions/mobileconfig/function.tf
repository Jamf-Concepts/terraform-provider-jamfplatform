# Build a complete .mobileconfig from native HCL payloads and deliver it via a
# configuration profile. Payloads use Apple's real payload keys; whole numbers
# become integers, fractional numbers reals, booleans/strings map directly,
# lists become arrays, nested objects become dictionaries.
resource "jamfplatform_pro_macos_configuration_profile" "workstation" {
  general = {
    name = "Workstation Baseline"
    payloads = provider::jamfplatform::mobileconfig({
      display_name = "Workstation Baseline"
      identifier   = "com.example.workstation"
      organization = "Example Org"
      payloads = [
        {
          PayloadType = "com.apple.dock"
          tilesize    = 48
          orientation = "left"
          autohide    = true
        },
        {
          PayloadType         = "com.apple.MCX"
          DisableGuestAccount = true
        },
      ]
    })
  }

  scope = {
    targets = {
      all_computers = true
    }
  }
}
