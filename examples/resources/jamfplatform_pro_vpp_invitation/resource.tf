# Self-Service VPP invitation scoped to a Jamf user group.
resource "jamfplatform_pro_vpp_invitation" "self_service" {
  name                = "Volume Purchasing Self Service"
  vpp_account_id      = "3"
  distribution_method = "Make available in Self Service only"

  # Automatically register users who have Managed Apple IDs (default true).
  auto_register_managed_users = true

  scope = {
    jss_user_group_ids = ["1"]

    exclusions = {
      # Directory-service (LDAP) groups are matched by name.
      directory_service_user_group_names = ["LDAP Admins"]
    }
  }
}

# Email-distribution VPP invitation. The sender / subject / message fields are
# required when distribution_method is "Send emails".
resource "jamfplatform_pro_vpp_invitation" "email" {
  name                 = "Volume Purchasing Email Invite"
  vpp_account_id       = "3"
  distribution_method  = "Send emails"
  sender_name          = "IT Support"
  sender_email_address = "it-support@example.com"
  subject              = "Register with Volume Purchasing"
  # %@ is replaced with the registration URL.
  message       = "Click the link to register with Volume Purchasing:\n\n%@"
  require_login = true

  scope = {
    all_jss_users = true
  }
}

# The per-user registration status is server-tracked and read-only.
output "vpp_invitation_usages" {
  value = jamfplatform_pro_vpp_invitation.self_service.invitation_usages
}
