# Lists every access policy application on the tenant. Jamf Security Cloud exposes no
# filter parameters for applications, so this list resource takes no configuration.
list "jamfplatform_security_cloud_ztna_app" "all" {
  provider         = jamfplatform
  include_resource = true

  config {}
}
