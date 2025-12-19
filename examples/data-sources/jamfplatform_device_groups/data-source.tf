# Filter to get all static computer groups
data "jamfplatform_device_groups" "static_computer_groups" {
  filter {
    selector = "deviceType"
    argument = "COMPUTER"
  }
  filter {
    join_with = "and"
    selector  = "groupType"
    argument  = "STATIC"
  }
}

# Filter to get all smart groups where the name starts with a specific substring
data "jamfplatform_device_groups" "smart_groups_name_wildcard_match" {
  filter {
    selector = "name"
    argument = "My Group*"
  }
  filter {
    join_with = "and"
    selector  = "groupType"
    argument  = "SMART"
  }
}

# Filter to get all device groups where the name does not contain a specific substring
data "jamfplatform_device_groups" "name_does_not_contain" {
  filter {
    selector = "name"
    operator = "!="
    argument = "*My Group*"
  }
}
