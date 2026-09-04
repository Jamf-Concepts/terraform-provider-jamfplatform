# Minimal macOS configuration profile: name + payload only.
resource "jamfplatform_pro_macos_configuration_profile" "minimal" {
  general = {
    name     = "Minimal Notifications Profile"
    payloads = file("${path.module}/minimal_notifications.mobileconfig")
  }
}

# Profile scoped to all computers with a per-computer exclusion. The payload
# is a Privacy Preferences Policy Control (PPPC) profile permitting
# Accessibility access for a specific app.
resource "jamfplatform_pro_macos_configuration_profile" "pppc_all_computers" {
  general = {
    name                = "PPPC – Accessibility"
    description         = "Allows Accessibility for Example.app"
    level               = "Computer Level"
    distribution_method = "Install Automatically"
    user_removable      = false
    redeploy_on_update  = "Newly Assigned"
    payloads            = file("${path.module}/pppc.mobileconfig")
  }

  scope = {
    targets = {
      all_computers = true
    }
    exclusions = {
      computer_ids = ["28"]
    }
  }
}

# Self Service profile with notifications + a category assignment. The Self
# Service tab requires `distribution_method = "Make Available in Self Service"`.
resource "jamfplatform_pro_macos_configuration_profile" "self_service" {
  general = {
    name                = "Self Service – Custom Wallpaper"
    distribution_method = "Make Available in Self Service"
    payloads            = file("${path.module}/wallpaper.mobileconfig")
  }

  scope = {
    targets = {
      all_computers = true
    }
  }

  self_service = {
    self_service_display_name     = "Custom Wallpaper"
    install_button_text           = "Install"
    self_service_description      = "Sets the desktop wallpaper to the corporate brand."
    ensure_users_view_description = true
    feature_on_main_page          = true
    display_notifications         = true
    notification_location         = "Self Service and Notification Center"
    notification_subject          = "Wallpaper available"
    notification_message          = "Install Custom Wallpaper from Self Service."
    removal_disallowed            = "Never"
    categories = [
      {
        id         = "64"
        display_in = true
        feature_in = true
      },
    ]
  }
}
