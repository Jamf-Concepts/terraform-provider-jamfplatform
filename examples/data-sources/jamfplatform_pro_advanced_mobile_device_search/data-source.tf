# Look up an advanced mobile device search by ID.
data "jamfplatform_pro_advanced_mobile_device_search" "by_id" {
  id = "478"
}

# Look up an advanced mobile device search by exact name.
data "jamfplatform_pro_advanced_mobile_device_search" "by_name" {
  name = "Unmanaged supervised iPads"
}
