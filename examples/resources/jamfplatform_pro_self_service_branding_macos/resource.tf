# Self Service macOS branding (Settings > Self Service > Branding > macOS Branding).
# Singleton — one configuration per tenant. All fields optional.
resource "jamfplatform_pro_self_service_branding_image" "icon" {
  image_file_source = "./self-service-icon.png"
}

resource "jamfplatform_pro_self_service_branding_image" "banner" {
  image_file_source = "./self-service-banner.png"
}

resource "jamfplatform_pro_self_service_branding_macos" "this" {
  application_header   = "Acme Self Service"
  sidebar_heading      = "Acme"
  sidebar_subheading   = "IT Self Service"
  home_page_heading    = "Welcome to Acme Self Service"
  home_page_subheading = "Install approved apps and run maintenance tasks"
  icon_id              = tonumber(jamfplatform_pro_self_service_branding_image.icon.id)
  banner_image_id      = tonumber(jamfplatform_pro_self_service_branding_image.banner.id)
}
