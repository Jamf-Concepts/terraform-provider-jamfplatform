# List every App Store Mac app in the tenant, optionally filtered by a
# case-insensitive name substring.
list "jamfplatform_pro_mac_app_store_app" "all" {
  provider = jamfplatform

  config {
    filter = {
      name_substring = "iMovie"
    }
  }
}
