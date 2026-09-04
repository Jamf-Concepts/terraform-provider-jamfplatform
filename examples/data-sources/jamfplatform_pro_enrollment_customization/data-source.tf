# Look up a Jamf Pro enrollment customization by ID or by exact display name.
# Exactly one of `id` or `display_name` must be supplied. Jamf Pro does not
# enforce unique display names; the lookup errors when the name matches more
# than one record.

data "jamfplatform_pro_enrollment_customization" "by_id" {
  id = "1"
}

data "jamfplatform_pro_enrollment_customization" "by_name" {
  display_name = "Welcome flow"
}

output "welcome_description" {
  value = data.jamfplatform_pro_enrollment_customization.by_name.description
}

output "welcome_icon_url" {
  value = data.jamfplatform_pro_enrollment_customization.by_name.branding_settings.icon_url
}
