# A user-initiated URL computer enrollment invitation with an SSH management
# account and a finite expiration. The invitation cannot be updated in place:
# changing any attribute forces Terraform to destroy and recreate it (which
# mints a new invitation code).
resource "jamfplatform_pro_computer_invitation" "url" {
  invitation_type = "USER_INITIATED_URL"

  # Either "Unlimited" or "yyyy-MM-dd HH:mm:ss" in the Jamf Pro server timezone.
  expiration_date = "2030-12-31 23:59:00"

  multiple_uses_allowed         = true
  keep_existing_site_membership = false

  # SSH management account provisioned on enrolled computers.
  create_account_if_does_not_exist = true
  hide_account                     = true
  lock_down_ssh                    = true
  ssh_username                     = "jamfmgmt"

  # ssh_password is WriteOnly: sent to Jamf Pro but never stored in state.
  # Bump ssh_password_wo_version to rotate (which forces a replace here, since
  # the endpoint has no update operation).
  ssh_password            = "change-me-to-a-strong-secret"
  ssh_password_wo_version = 1
}

# A non-expiring email invitation assigned to a site by ID. An SSH management
# account username is required by Jamf Pro on every computer invitation.
resource "jamfplatform_pro_computer_invitation" "email" {
  invitation_type = "USER_INITIATED_EMAIL"
  expiration_date = "Unlimited"
  ssh_username    = "jamfmgmt"

  # Jamf Pro site ID; use "-1" for "None". Set a real site ID to assign a site.
  enroll_into_site_id = "-1"
}
