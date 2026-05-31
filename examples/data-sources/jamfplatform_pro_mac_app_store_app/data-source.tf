# Look up an App Store Mac app by exact name.
data "jamfplatform_pro_mac_app_store_app" "by_name" {
  name = "iMovie"
}

output "imovie_id" {
  value = data.jamfplatform_pro_mac_app_store_app.by_name.id
}

# Look up an App Store Mac app by ID.
data "jamfplatform_pro_mac_app_store_app" "by_id" {
  id = "84"
}
