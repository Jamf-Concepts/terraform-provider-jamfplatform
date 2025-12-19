data "device_group" "test_group" {
  count = length(data.device_groups.test_all_groups.device_groups) > 0 ? 1 : 0
  id    = "data.device_groups.test_all_groups.device_groups[0].id"
}
