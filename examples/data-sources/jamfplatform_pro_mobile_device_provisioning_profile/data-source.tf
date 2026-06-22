data "jamfplatform_pro_mobile_device_provisioning_profile" "by_id" {
  id = "3"
}

data "jamfplatform_pro_mobile_device_provisioning_profile" "by_name" {
  name = "In-House App Profile"
}

data "jamfplatform_pro_mobile_device_provisioning_profile" "by_uuid" {
  uuid = "beeb6fc5-416f-40ba-bb19-b7a1714f8d83"
}

output "provisioning_profile_by_name_uuid" {
  value = data.jamfplatform_pro_mobile_device_provisioning_profile.by_name.uuid
}
