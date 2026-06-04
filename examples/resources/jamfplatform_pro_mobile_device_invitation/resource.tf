# A user-initiated URL mobile device enrollment invitation with a finite
# expiration. The invitation cannot be updated in place: changing any attribute
# forces Terraform to destroy and recreate it (which mints a new invitation
# code).
resource "jamfplatform_pro_mobile_device_invitation" "url" {
  invitation_type = "USER_INITIATED_URL"

  # Either "Unlimited" or "yyyy-MM-dd HH:mm:ss" in the Jamf Pro server timezone.
  expiration_date = "2030-12-31 23:59:00"

  multiple_uses_allowed         = true
  require_login                 = true
  keep_existing_site_membership = false
}

# A non-expiring email invitation assigned to a site by ID and delivered to a
# recipient.
resource "jamfplatform_pro_mobile_device_invitation" "email" {
  invitation_type = "USER_INITIATED_EMAIL"
  expiration_date = "Unlimited"

  # Jamf Pro site ID; use "-1" for "None". Set a real site ID to assign a site.
  enroll_into_site_id = "-1"

  require_login = true
  subject       = "Enroll your mobile device"
  message       = "Tap the link to enroll your device with Jamf."
  reply_to      = "noreply@example.com"
  sent_from     = "it@example.com"
  sent_to       = "user@example.com"
}
