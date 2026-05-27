# List every Jamf Pro policy in the tenant, optionally filtered by a
# case-insensitive name substring.
list "jamfplatform_pro_policy" "all" {
  provider = jamfplatform

  config {
    filter {
      name_substring = "tf-acc-"
    }
  }
}
