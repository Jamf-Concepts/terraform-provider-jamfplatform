# Look up a mobile device invitation by its numeric ID.
data "jamfplatform_pro_mobile_device_invitation" "example_by_id" {
  id = "42"
}

# Look up a mobile device invitation by its Jamf Pro-generated invitation code (the
# admin UI "Invitation ID"). Exactly one of id or invitation must be supplied.
data "jamfplatform_pro_mobile_device_invitation" "example_by_invitation" {
  invitation = "308000000000000000000000000000000000001"
}

output "mobile_device_invitation_example_by_id" {
  value = data.jamfplatform_pro_mobile_device_invitation.example_by_id
}

output "mobile_device_invitation_example_by_invitation" {
  value = data.jamfplatform_pro_mobile_device_invitation.example_by_invitation
}
