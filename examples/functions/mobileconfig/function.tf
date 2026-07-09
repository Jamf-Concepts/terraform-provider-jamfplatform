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

# A real-world, multi-payload example: everything the open-source SAP
# Privileges app needs in one profile. This mixes three different payload
# types — the case the generic mobileconfig function exists for, since a
# single-payload helper like mcx_forced_payload cannot express it.
resource "jamfplatform_pro_macos_configuration_profile" "privileges" {
  general = {
    name = "Privileges"
    payloads = provider::jamfplatform::mobileconfig({
      display_name       = "Privileges"
      identifier         = "com.example.privileges"
      organization       = "Example Org"
      scope              = "System"
      removal_disallowed = true

      payloads = [
        {
          # Managed Login Items: approve the app's background components by
          # signing Team ID so users can't disable them in System Settings.
          PayloadType        = "com.apple.servicemanagement"
          PayloadDisplayName = "Managed Login Items"
          PayloadDescription = "Background Service Management for Privileges"
          Rules = [
            {
              Comment   = "Approves Privileges and its components"
              RuleType  = "TeamIdentifier"
              RuleValue = "7R5ZEU67FQ"
            },
          ]
        },
        {
          # Pre-approve user notifications for both of the app's bundles.
          PayloadType        = "com.apple.notificationsettings"
          PayloadDisplayName = "Notifications Payload"
          NotificationSettings = [
            for bundle in ["corp.sap.privileges", "corp.sap.privileges.agent"] : {
              AlertType                = 1
              BadgesEnabled            = false
              BundleIdentifier         = bundle
              NotificationsEnabled     = true
              ShowInLockScreen         = false
              ShowInNotificationCenter = false
              SoundsEnabled            = false
            }
          ]
        },
        {
          # The app's managed preferences, wrapped in the MCX "Forced"
          # envelope by hand (PayloadContent -> <domain> -> Forced ->
          # mcx_preference_settings). Deploying the keys under a bare
          # app-domain PayloadType would also work on-device, but Jamf Pro's
          # admin UI cannot render that shape; the MCX envelope shows the
          # settings under Application & Custom Settings. For a profile that
          # carries ONLY managed preferences, prefer the mcx_forced_payload
          # function, which builds this envelope for you.
          PayloadType        = "com.apple.ManagedClient.preferences"
          PayloadDisplayName = "Custom Settings"
          PayloadContent = {
            "corp.sap.privileges" = {
              Forced = [
                {
                  mcx_preference_settings = {
                    DockToggleTimeout              = 10
                    ExpirationIntervalMax          = 10
                    HelpButtonCustomURL            = "https://wiki.example.com/privileges-help"
                    HideSettingsButton             = true
                    RequireAuthentication          = true
                    RequireBiometricAuthentication = true
                    RevokeAtLoginExcludedUsers     = ["localadmin", "labadmin"]
                    RevokePrivilegesAtLogin        = true
                  }
                },
              ]
            }
          }
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
