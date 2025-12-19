data "jamfplatform_device" "test_device" {
  count = length(data.jamfplatform_devices.test_all_devices.devices) > 0 ? 1 : 0
  id    = data.jamfplatform_devices.test_all_devices.devices[0].id
}
