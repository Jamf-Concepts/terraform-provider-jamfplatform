# Upload a signed in-house (enterprise) .mobileprovision profile.
# An uploaded profile cannot be modified in place: changing name or profile_data
# replaces the profile (Terraform deletes and recreates it). display_name is
# computed: Jamf Pro sets it to match name.
resource "jamfplatform_pro_mobile_device_provisioning_profile" "in_house" {
  name = "In-House App Profile"

  # Base64-encoded signed .mobileprovision. Jamf Pro parses the UUID and
  # expiration out of it; those land in the computed attributes below.
  profile_data = filebase64("${path.module}/InHouse.mobileprovision")
}

output "in_house_profile_uuid" {
  value = jamfplatform_pro_mobile_device_provisioning_profile.in_house.uuid
}

output "in_house_profile_expiration" {
  value = jamfplatform_pro_mobile_device_provisioning_profile.in_house.expiration_date_utc
}
