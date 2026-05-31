# Look up a mobile device app by exact name.
data "jamfplatform_pro_mobile_device_app" "by_name" {
  name = "Maps"
}

output "maps_id" {
  value = data.jamfplatform_pro_mobile_device_app.by_name.id
}

# Look up a mobile device app by ID.
data "jamfplatform_pro_mobile_device_app" "by_id" {
  id = "122"
}
