# List every webhook in the tenant, optionally filtered by a case-insensitive
# name substring.
list "jamfplatform_pro_webhook" "all" {
  provider = jamfplatform

  config {
    filter = {
      name_substring = "check-in"
    }
  }
}
