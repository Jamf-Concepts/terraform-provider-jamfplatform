# Lists every grouped gateway on the tenant. Jamf Security Cloud exposes no filter
# parameters for grouped gateways, so this list resource takes no configuration.
list "jamfplatform_security_cloud_ztna_grouped_gateway" "all" {
  provider         = jamfplatform
  include_resource = true

  config {}
}
