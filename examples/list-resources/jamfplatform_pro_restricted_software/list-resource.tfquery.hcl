# List every restricted software record in the tenant, optionally filtered by a
# case-insensitive name substring.
list "jamfplatform_pro_restricted_software" "all" {
  provider = jamfplatform

  config {
    filter = {
      name_substring = "Block"
    }
  }
}
