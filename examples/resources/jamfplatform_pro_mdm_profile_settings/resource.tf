resource "jamfplatform_pro_mdm_profile_settings" "this" {
  auto_renew_computer_profile_when_ca_renewed      = true
  auto_renew_computer_profile_before_expiry        = true
  computer_profile_expiration_limit_days           = 180
  auto_renew_mobile_device_profile_when_ca_renewed = true
  auto_renew_mobile_device_profile_before_expiry   = true
  mobile_device_profile_expiration_limit_days      = 180
}
