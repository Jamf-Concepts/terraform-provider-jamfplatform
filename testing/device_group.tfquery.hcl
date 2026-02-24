list "jamfplatform_device_group" "test_device_group" {
  provider = jamfplatform

  config {

    filter = [
      {
        selector = "deviceType"
        argument = "COMPUTER"
      },
      {
        join_with = "and"
        selector  = "groupType"
        argument  = "STATIC"
      }
    ]
  }
}
