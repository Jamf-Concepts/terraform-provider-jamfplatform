# Lists every device group on the tenant that Terraform can manage. Jamf Security
# Cloud exposes no filter parameters for groups, so this list resource takes no
# configuration. The built-in group is not listed — it has no identifier, so it
# cannot be imported.
list "jamfplatform_security_cloud_device_group" "all" {
  provider         = jamfplatform
  include_resource = true

  config {}
}
