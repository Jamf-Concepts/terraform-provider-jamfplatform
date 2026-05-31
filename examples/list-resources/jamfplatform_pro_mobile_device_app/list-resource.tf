# List every mobile device app in the tenant, optionally filtered by a
# case-insensitive name substring.
list "jamfplatform_pro_mobile_device_app" "all" {
  provider = jamfplatform

  config {
    filter = {
      name_substring = "Maps"
    }
  }
}
