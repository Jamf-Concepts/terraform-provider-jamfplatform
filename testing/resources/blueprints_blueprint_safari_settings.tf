resource "jamfplatform_blueprints_blueprint" "test_safari_settings" {
  name        = "Terraform Test Safari Settings ${var.test_id}"
  description = "Managed by Terraform"

  device_groups = [data.jamfpro_group.test_target_computer_group.group_platform_id]

  safari_settings {
    accept_cookies                  = "VisitedWebsites"
    allow_disabling_fraud_warning   = false
    allow_history_clearing          = false
    allow_javascript                = true
    allow_private_browsing          = false
    allow_popups                    = false
    allow_summary                   = true
    new_tab_start_page_type         = "Home"
    new_tab_start_page_homepage_url = "https://company.com"
  }
}
