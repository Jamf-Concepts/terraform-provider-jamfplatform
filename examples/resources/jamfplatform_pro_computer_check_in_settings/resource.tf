resource "jamfplatform_pro_computer_check_in_settings" "this" {
  check_in_frequency                  = 15
  create_startup_script               = false
  startup_log                         = true
  startup_policies                    = true
  startup_ssh                         = false
  create_login_hook                   = false
  login_hook_log                      = true
  login_hook_policies                 = true
  allow_network_state_change_triggers = false
}
