# The tenant's configured search domain. Takes no arguments, since there is one
# per tenant. Reading it when none is configured is an error.
data "jamfplatform_security_cloud_dns_search_domain" "current" {}

output "search_domain" {
  value = data.jamfplatform_security_cloud_dns_search_domain.current.domain_name
}
