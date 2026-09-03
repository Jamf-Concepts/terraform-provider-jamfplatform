# Self Service iOS & iPadOS branding (Settings > Self Service > Branding > iOS & iPadOS Branding).
# One configuration per tenant. main_header + colour codes are required.
resource "jamfplatform_pro_self_service_branding_image" "icon" {
  image_file_source = "./self-service-icon.png"
}

resource "jamfplatform_pro_self_service_branding_ios" "this" {
  main_header                  = "Acme Self Service"
  branding_name_color_code     = "000000" # Main Header text colour (6-digit hex, no '#')
  header_background_color_code = "FFFFFF" # Header background colour
  menu_icon_color_code         = "007AFF" # Menu icon colour
  status_bar_text_color        = "dark"   # "light" or "dark"
  icon_id                      = tonumber(jamfplatform_pro_self_service_branding_image.icon.id)
}
