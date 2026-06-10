# Upload a branding image (icon or banner) for Self Service branding.
# The resulting id is referenced by the macOS / iOS branding resources.
# Local file — provider hashes bytes on every plan; replaces on content change.
resource "jamfplatform_pro_self_service_branding_image" "icon" {
  image_file_source = "./self-service-icon.png" # recommended 180x180 PNG/GIF
}

resource "jamfplatform_pro_self_service_branding_image" "banner" {
  image_file_source = "./self-service-banner.png" # recommended 1500x235 PNG/GIF
}

# URL source — provider downloads on every plan; replaces on upstream change.
# resource "jamfplatform_pro_self_service_branding_image" "from_url" {
#   image_file_source = "https://cdn.example.com/self-service-icon.png"
# }
