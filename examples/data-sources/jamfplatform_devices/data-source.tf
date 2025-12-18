data "jamfplatform_devices" "all" {
}

data "jamfplatform_devices" "example_by_model" {
  filter = "model==\"MacBook Pro\""
}
