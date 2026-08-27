# Lists every dedicated gateway on the tenant. Jamf Security Cloud exposes no
# filter parameters for gateways, so this list resource takes no configuration.
#
# Generated configuration for an IPSec gateway omits the pre-shared key, which
# Jamf Security Cloud never returns — fill it in before applying.
list "jamfplatform_security_cloud_ztna_gateway" "all" {
  provider         = jamfplatform
  include_resource = true

  config {}
}
