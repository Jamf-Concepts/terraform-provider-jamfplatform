resource "jamfplatform_pro_user_initiated_enrollment_settings" "this" {
  # General tab
  skip_certificate_installation = false
  restrict_reenrollment         = true
  signing_mdm_profile_enabled   = false

  # Computers tab
  enable_computer_enrollment             = true
  create_management_account              = true
  management_username                    = "lapsadmin"
  hide_management_account                = true
  allow_ssh_only_management_account      = false
  ensure_ssh_running                     = false
  launch_self_service                    = true
  sign_quickadd_package                  = false
  account_driven_device_enrollment_macos = false

  # Devices tab
  profile_driven_enrollment_via_url_institutional = true
  profile_driven_enrollment_via_url_personal      = false
  account_driven_user_enrollment                  = false
  account_driven_user_enrollment_visionos         = false
  merge_managed_apple_account_usernames           = false
  account_driven_device_enrollment_ios            = false
  account_driven_device_enrollment_visionos       = false

  # Directory-service Access Groups (Access tab).
  # Supply each group by name + ldap_server_id; the provider resolves the
  # directory's group id for you (directory_service_group_id is Computed). The
  # built-in "All Directory Service Users" group is left untouched when omitted;
  # declare it (with ldap_server_id = "-1") to edit its toggles.
  access_group = [
    {
      name                          = "Jamf Pro Admins"
      ldap_server_id                = "7"
      enterprise_enrollment_enabled = true
    },
  ]

  # Per-language enrollment messaging (Messaging tab), keyed by ISO 639-1
  # language code. Set only the fields you want to override; unset fields are
  # seeded from the current English messaging when a language is first added,
  # and otherwise left at their current server value. The built-in English
  # language always exists and is never removed. Set the "en" key to edit its
  # text, or omit it to leave it untouched.
  messaging_languages = {
    fr = {
      page_title                = "Inscrivez votre appareil"
      login_button_text         = "Connexion"
      enroll_device_button_name = "Inscrire"
    }
  }
}

# To use a third-party MDM signing certificate, set
# signing_mdm_profile_enabled = true and supply the keystore:
#
# resource "jamfplatform_pro_user_initiated_enrollment_settings" "with_cert" {
#   signing_mdm_profile_enabled = true
#   mdm_signing_certificate = {
#     keystore_file                = filebase64("mdm-signing.p12")
#     keystore_file_name           = "mdm-signing.p12"
#     keystore_password            = var.mdm_keystore_password # WriteOnly
#     keystore_password_wo_version = 1
#   }
# }
