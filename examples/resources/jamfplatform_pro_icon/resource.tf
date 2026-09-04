# Local file. Provider hashes bytes on every plan. Stable unless file changes.
resource "jamfplatform_pro_icon" "from_local" {
  icon_file_source = "./icon.png"
}

# URL. Provider downloads on every plan. Replaces when upstream content changes.
# Useful for tracking App Store / vendor CDN icons.
resource "jamfplatform_pro_icon" "from_url" {
  icon_file_source = "https://is1-ssl.mzstatic.com/image/thumb/.../512x512bb.jpg"
}
