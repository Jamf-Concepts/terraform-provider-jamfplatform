# Asks Jamf to re-check the ownership record for a claimed domain. Run it once
# the TXT record is published and has had time to propagate.
#
# Do NOT wire this action to the claiming resource's own lifecycle — no
# action_trigger with events = [after_create] on jamfplatform_account_sso_domain.
# Jamf allows one check every five minutes per domain and claiming a domain
# starts that clock, so a check in the same run as the claim is refused every
# single time, and the refusal fails the apply. Invoke it on a later run
# instead, naming the action on the command line:
#
#   terraform apply -invoke='action.jamfplatform_account_sso_domain_verify.corp'
#
# Two more things are worth knowing:
#
#   - A check that finds nothing is reported as a failure, so a green run means
#     the domain really is verified.
#   - Each check moves the domain's fourteen-day verification deadline, whether
#     it succeeds or not, so it is not free to run repeatedly.
#
# The two blocks below illustrate the two configuration forms, not two checks to
# run together: under the five-minute limit, invoking the second against the
# same domain right after the first is refused by construction. Pick one.

# Naming the domain is usually what you want: it is the identifier shown in Jamf
# Account.
action "jamfplatform_account_sso_domain_verify" "corp" {
  config {
    domain = jamfplatform_account_sso_domain.corp.domain
  }
}

# Pass domain_id instead to skip the name lookup, which needs one permission
# fewer.
action "jamfplatform_account_sso_domain_verify" "by_id" {
  config {
    domain_id = jamfplatform_account_sso_domain.corp.id
  }
}

resource "jamfplatform_account_sso_domain" "corp" {
  domain = "example.com"
}
