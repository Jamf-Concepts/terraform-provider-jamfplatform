action "jamfplatform_pro_retry_app_installer_installations" "retry" {
  config {
    deployment_id = jamfplatform_pro_app_installer.chrome.id

    # Omit computer_ids to retry every failed installation in the deployment.
    computer_ids = ["456", "789"]
  }
}

# Retry after any change to the deployment. With nothing to retry the action
# warns instead of failing, so this is safe to leave wired up.
resource "terraform_data" "retry_on_change" {
  input = jamfplatform_pro_app_installer.chrome.selected_version

  lifecycle {
    action_trigger {
      events  = [after_create, after_update]
      actions = [action.jamfplatform_pro_retry_app_installer_installations.retry]
    }
  }
}
