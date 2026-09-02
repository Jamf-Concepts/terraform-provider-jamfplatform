# Asks Jamf to re-check the ownership record for a claimed domain. Run it once
# the TXT record is published and has had time to propagate.
#
# Three things are worth knowing before wiring this into a pipeline:
#
#   - Jamf allows one check every five minutes per domain, and claiming a domain
#     starts that clock — so a check immediately after a claim is refused. Wait,
#     then run it.
#   - A check that finds nothing is reported as a failure, so a green run means
#     the domain really is verified.
#   - Each check moves the domain's fourteen-day verification deadline, whether
#     it succeeds or not, so it is not free to run repeatedly.
action "jamfplatform_account_sso_domain_verify" "corp" {
  config {
    domain = jamfplatform_account_sso_domain.corp.domain
  }
}

# Naming the domain is usually what you want: it is the identifier shown in Jamf
# Account, and referencing the resource attribute also orders the check after the
# claim. Pass domain_id instead to skip the name lookup, which needs one
# permission fewer.
action "jamfplatform_account_sso_domain_verify" "by_id" {
  config {
    domain_id = jamfplatform_account_sso_domain.corp.id
  }
}

resource "jamfplatform_account_sso_domain" "corp" {
  domain = "example.com"
}
