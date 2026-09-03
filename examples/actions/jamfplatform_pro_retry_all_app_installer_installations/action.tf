# Retries every failed App Installer installation in the tenant, across all
# deployments. Scope one deployment with
# jamfplatform_pro_retry_app_installer_installations instead.
action "jamfplatform_pro_retry_all_app_installer_installations" "retry_all" {
  config {}
}

resource "terraform_data" "nightly_retry" {
  input = timestamp()

  lifecycle {
    action_trigger {
      events  = [after_update]
      actions = [action.jamfplatform_pro_retry_all_app_installer_installations.retry_all]
    }
  }
}
