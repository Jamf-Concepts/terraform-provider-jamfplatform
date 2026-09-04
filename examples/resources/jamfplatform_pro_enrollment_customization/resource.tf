# Manages a Jamf Pro enrollment customization: the parent record carrying the
# branding palette plus any combination of text, LDAP, and SSO authentication
# panes shown to users during enrollment.
#
# At most one authentication pane (LDAP or SSO) may be configured per
# customization; the two are mutually exclusive.
#
# The icon may be supplied either as a local file path (`icon_source`,
# re-uploaded automatically when its bytes change) or as a pre-uploaded URL
# (`branding_settings.icon_url`); the two are mutually exclusive.

resource "jamfplatform_pro_enrollment_customization" "welcome" {
  display_name = "Welcome flow"
  description  = "Default enrollment experience for staff devices"

  # Either set icon_source (local path) OR branding_settings.icon_url
  # (pre-uploaded URL), not both. Leave both unset to skip the icon.
  icon_source = "${path.module}/welcome.png"

  branding_settings = {
    body_text_color   = "333333"
    button_color      = "0066cc"
    button_text_color = "ffffff"
    background_color  = "ffffff"
  }

  text_panes = [
    {
      display_name         = "Welcome"
      rank                 = 0
      title                = "Welcome to Acme"
      body                 = "Thanks for joining. We'll get your Mac set up in a few steps."
      previous_button_text = "Back"
      next_button_text     = "Get started"
    },
    {
      display_name         = "Terms"
      rank                 = 1
      title                = "Acceptable use"
      body                 = "Use this device for company business only. Tap Accept to continue."
      subtext              = "You can review the full policy at any time."
      previous_button_text = "Back"
      next_button_text     = "Accept"
    },
  ]

  # Exactly one of ldap_panes or sso_panes (or neither) may be supplied.
  sso_panes = [
    {
      display_name                   = "Sign in with SSO"
      rank                           = 2
      enrollment_access              = "specific_group"
      access_group_name              = "Enrollers"
      pass_user_info_to_jamf_connect = true
      account_name_attribute         = "shortName"
      account_full_name_attribute    = "fullName"
    },
  ]
}

output "enrollment_customization_id" {
  value = jamfplatform_pro_enrollment_customization.welcome.id
}
