# Lists every custom DNS zone on the tenant. Jamf Security Cloud exposes no
# filter parameters for zones, so this list resource takes no configuration.
list "jamfplatform_security_cloud_dns_zone" "all" {
  provider         = jamfplatform
  include_resource = true

  config {}
}
