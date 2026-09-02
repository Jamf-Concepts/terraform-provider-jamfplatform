# Every connection, for generating configuration from what already exists. Jamf
# returns the whole set, so there is nothing to filter on.
#
# Connections Terraform cannot manage are left out of the results rather than
# offered as imports that no apply could reconcile.
list "jamfplatform_account_sso_connection" "all" {
  provider = jamfplatform

  config {}
}
