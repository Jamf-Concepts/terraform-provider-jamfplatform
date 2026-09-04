# Every claimed domain, for generating configuration from what already exists.
# Jamf Account returns the whole set — there is nothing to filter on here.
list "jamfplatform_account_sso_domain" "all" {
  provider = jamfplatform

  config {}
}
