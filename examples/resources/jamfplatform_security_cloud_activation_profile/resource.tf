# An activation profile is the enrollment credential end users activate Jamf Trust
# against. Creating one mints an activation code, which you either distribute as a
# link or hand to a UEM.

# Terraform manages a small part of an activation profile. The end user
# application, authentication and identity provider, block pages, expiration date
# and device location settings are configured in Jamf Security Cloud, and a
# profile created here takes the Jamf Security Cloud default for each of them.

resource "jamfplatform_security_cloud_activation_profile" "content_filtering" {
  name      = "Content Filtering — iOS"
  platforms = ["ios"]

  capabilities = {
    content_controls = true
  }
}

# Adding devices to a group as they enrol keeps policy assignment declarative.
# Jamf Security Cloud does not check the group exists: an identifier that
# matches nothing leaves enrolling devices in no group at all, silently.
resource "jamfplatform_security_cloud_device_group" "field_staff" {
  name = "Field Staff"
}

resource "jamfplatform_security_cloud_activation_profile" "field_staff" {
  name      = "Field Staff"
  platforms = ["ios", "mac"]

  device_group_id = jamfplatform_security_cloud_device_group.field_staff.id

  capabilities = {
    content_controls = true
    network_security = true
    note             = "Managed by Terraform"
  }
}

# Pausing stops a profile accepting new enrolments without destroying it, and is
# the only change that does not mint a new activation code. Every other attribute
# replaces the profile, which invalidates any code already distributed.
resource "jamfplatform_security_cloud_activation_profile" "seasonal" {
  name      = "Seasonal Contractors"
  platforms = ["ios"]
  paused    = true

  capabilities = {
    network_security = true
  }
}

# The activation code is what reaches a device, as a link or deployed to a UEM.
# It is a credential: anyone holding it can enrol a device, so the attribute is
# sensitive and any output carrying it must be marked sensitive too.
output "field_staff_activation_code" {
  description = "Activation code end users enrol against."
  value       = jamfplatform_security_cloud_activation_profile.field_staff.id
  sensitive   = true
}
