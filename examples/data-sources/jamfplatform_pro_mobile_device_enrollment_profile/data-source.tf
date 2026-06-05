data "jamfplatform_pro_mobile_device_enrollment_profile" "by_id" {
  id = "61"
}

data "jamfplatform_pro_mobile_device_enrollment_profile" "by_name" {
  name = "Configurator Enrollment Profile"
}

data "jamfplatform_pro_mobile_device_enrollment_profile" "by_invitation" {
  invitation = "138277457037961032316766183186860280252"
}

output "enrollment_profile_uuid" {
  value = data.jamfplatform_pro_mobile_device_enrollment_profile.by_name.uuid
}
