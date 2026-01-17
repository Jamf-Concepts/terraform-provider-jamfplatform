action "jamfplatform_device_erase" "return_device_to_service" {
  config {
    device_id = "uuid-of-device-to-erase"

    # Optional erase request fields – remove or adjust as needed.
    preserve_data_plan       = true
    disallow_proximity_setup = false
    clear_activation_lock    = true
    return_to_service        = true
    pin                      = null
  }
}
