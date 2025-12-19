data "jamfplatform_device_group" "test_group" {
  count = length(data.jamfplatform_device_groups.test_all_groups.device_groups) > 0 ? 1 : 0
  id    = data.jamfplatform_device_groups.test_all_groups.device_groups[0].id
}
