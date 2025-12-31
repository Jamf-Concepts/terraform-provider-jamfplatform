# Query to list smart mobile device groups
list "jamfplatform_device_group" "smart_mobile_device_groups" {
  provider = jamfplatform

  config {

    filter {
      selector = "deviceType"
      argument = "MOBILE"
    }

    filter {
      join_with = "and"
      selector  = "groupType"
      argument  = "SMART"
    }
  }
}

# Query to list static computer groups
list "jamfplatform_device_group" "static_computer_groups" {
  provider = jamfplatform

  config {

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
}
