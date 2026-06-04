# Look up a computer invitation by its numeric ID.
data "jamfplatform_pro_computer_invitation" "example_by_id" {
  id = "42"
}

# Look up a computer invitation by its Jamf Pro-generated invitation code (the
# admin UI "Invitation ID"). Exactly one of id or invitation must be supplied.
data "jamfplatform_pro_computer_invitation" "example_by_invitation" {
  invitation = "308000000000000000000000000000000000001"
}

output "computer_invitation_example_by_id" {
  value = data.jamfplatform_pro_computer_invitation.example_by_id
}

output "computer_invitation_example_by_invitation" {
  value = data.jamfplatform_pro_computer_invitation.example_by_invitation
}
