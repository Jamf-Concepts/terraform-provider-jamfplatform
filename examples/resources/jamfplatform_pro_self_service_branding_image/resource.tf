# Upload a branding image (icon or banner) for Self Service branding.
# The resulting id is referenced by the macOS / iOS branding resources.

# Local file. The provider hashes the bytes on every plan, so editing the file
# replaces the image and the next plan shows it.
resource "jamfplatform_pro_self_service_branding_image" "icon" {
  image_file_source = "./self-service-icon.png" # recommended 180x180 PNG/GIF
}

resource "jamfplatform_pro_self_service_branding_image" "banner" {
  image_file_source = "./self-service-banner.png" # recommended 1500x235 PNG/GIF
}

# URL source. The provider reads the bytes during apply and compares the URL
# string at plan time. Re-point it to replace the image. The provider cannot see
# a new file published behind this URL.
# resource "jamfplatform_pro_self_service_branding_image" "from_url" {
#   image_file_source = "https://cdn.example.com/self-service-icon.png"
# }
