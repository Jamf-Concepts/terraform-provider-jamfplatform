# List every Jamf Pro mobile device enrollment invitation. Mobile device
# invitations carry no name, so this list resource takes no filter
# configuration. Note: the invitation list can lag newly created invitations,
# so a just-created invitation may not appear here immediately.
list "jamfplatform_pro_mobile_device_invitation" "all" {
  provider = jamfplatform
}
