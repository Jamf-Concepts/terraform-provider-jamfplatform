# The search domain completes an incomplete host name for apps that only accept
# short names: with this set, a user asking for "product" is directed to
# "product.corp.example.com".
#
# There is one search domain per tenant, so only one instance of this resource
# should exist in your configuration. Destroying it clears the search domain for the
# whole tenant.
resource "jamfplatform_security_cloud_dns_search_domain" "corp" {
  domain_name = "corp.example.com"
}
