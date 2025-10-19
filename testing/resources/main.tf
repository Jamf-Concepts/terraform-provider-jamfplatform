resource "jamfpro_smart_computer_group" "test_target_computer_group" {
  name = "Terraform Test Target Computer Group ${var.test_id}"
  criteria {
    name        = "Serial Number"
    search_type = "is"
    priority    = 0
    value       = "terraform-test"
    and_or      = "and"
  }
}

resource "jamfpro_smart_mobile_device_group" "test_target_mobile_device_group" {
  name = "Terraform Test Target Mobile Device Group ${var.test_id}"
  criteria {
    name        = "Serial Number"
    search_type = "is"
    priority    = 0
    value       = "terraform-test"
    and_or      = "and"
  }
}

data "jamfpro_group" "test_target_computer_group" {
  group_jamfpro_id = jamfpro_smart_computer_group.test_target_computer_group.id
  group_type       = "COMPUTER"
}

data "jamfpro_group" "test_target_mobile_device_group" {
  group_jamfpro_id = jamfpro_smart_mobile_device_group.test_target_mobile_device_group.id
  group_type       = "MOBILE"
}
