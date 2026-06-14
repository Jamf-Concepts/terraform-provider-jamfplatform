resource "jamfplatform_pro_app_installer_settings" "example" {
  deployment_settings = {
    batch_size       = 1000
    batch_frequency  = 60
    days             = ["MONDAY", "TUESDAY", "WEDNESDAY", "THURSDAY", "FRIDAY"]
    server_time_from = "08:00:00Z"
    server_time_to   = "17:00:00Z"
  }

  end_user_experience = {
    notification_frequency  = 2
    notification_message    = "An update is available for this app."
    update_deadline         = 24
    force_quit_message      = "Please save your work. This app will close shortly."
    force_quit_grace_period = 10
    update_complete_message = "The app has been updated successfully."
    relaunch                = true
    suppress                = false
  }
}
