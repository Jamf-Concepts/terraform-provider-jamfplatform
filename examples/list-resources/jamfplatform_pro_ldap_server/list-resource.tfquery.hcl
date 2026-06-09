# List Jamf Pro LDAP servers via `terraform query`. The classic /ldapservers
# endpoint has no server-side filtering, so name_substring is applied
# client-side (case-insensitive). Set include_resource = true to fetch the full
# record for each match (an extra GET per item).
list "jamfplatform_pro_ldap_server" "ad_servers" {
  provider         = jamfplatform
  include_resource = true

  config {
    filter = {
      name_substring = "AD"
    }
  }
}
