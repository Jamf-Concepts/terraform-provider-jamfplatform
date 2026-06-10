# Manage the Self Service for macOS app settings (Settings > Self Service > macOS).
# Singleton — one record per tenant; declare it once. Omitted fields keep their
# current Jamf Pro value.
resource "jamfplatform_pro_category" "self_service_home" {
  name     = "Featured"
  priority = 9
}

resource "jamfplatform_pro_self_service_macos_settings" "settings" {
  install_automatically  = true
  install_location       = "/Applications"
  notifications_enabled  = true
  bookmarks_display_name = "Bookmarks"

  # Require user login via Single Sign-On with FIDO2.
  login_method        = "Required"
  authentication_type = "Saml"
  fido2_enabled       = true

  # Open Self Service on a Browse category (-1 = All Items).
  default_landing_page     = "BROWSE"
  default_home_category_id = tonumber(jamfplatform_pro_category.self_service_home.id)
}
