# Local file. The provider hashes the bytes on every plan, so editing the file
# replaces the icon and the next plan shows it.
resource "jamfplatform_pro_icon" "from_local" {
  icon_file_source = "./icon.png"
}

# URL. The provider reads the bytes during apply and compares the URL string at
# plan time. Re-point it to replace the icon; a new image behind this URL is not
# something the provider can see.
resource "jamfplatform_pro_icon" "from_url" {
  icon_file_source = "https://is1-ssl.mzstatic.com/image/thumb/.../512x512bb.jpg"
}
