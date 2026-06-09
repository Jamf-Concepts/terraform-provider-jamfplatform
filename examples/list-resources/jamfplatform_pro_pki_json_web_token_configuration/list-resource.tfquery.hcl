# List every JSON Web Token configuration in the tenant (Jamf Pro allows at
# most one), optionally filtered by a case-insensitive name substring.
list "jamfplatform_pro_pki_json_web_token_configuration" "all" {
  provider = jamfplatform

  config {
    filter = {
      name_substring = "setup"
    }
  }
}
