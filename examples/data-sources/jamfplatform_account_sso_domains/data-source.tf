# Every domain the organization has claimed.
data "jamfplatform_account_sso_domains" "all" {}

# Domains still waiting on their ownership record, which is the usual reason a
# single sign-on connection cannot be created yet.
output "awaiting_verification" {
  value = [
    for d in data.jamfplatform_account_sso_domains.all.sso_domains :
    d.domain if d.verification_status != "VERIFIED" && d.verification_status != "MANUALLY_VERIFIED" && d.verification_status != "MS_VERIFIED"
  ]
}

# Domains owned by another organization and shared with this one. They can be
# used by a connection here, but not changed or removed here.
output "shared_in_from_elsewhere" {
  value = [for d in data.jamfplatform_account_sso_domains.all.sso_domains : d.domain if d.shared]
}
