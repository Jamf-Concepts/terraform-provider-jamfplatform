# List every Jamf Pro Automated Device Enrollment (ADE) instance.
list "jamfplatform_pro_automated_device_enrollment" "all" {
  provider = jamfplatform
}

# List ADE instances whose name contains the substring "prod"
# (case-insensitive). The list endpoint returns the full ADE shape per row,
# so include_resource = true does not trigger a follow-up GET per item.
list "jamfplatform_pro_automated_device_enrollment" "prod_instances" {
  provider = jamfplatform

  config {
    filter = {
      name_substring = "prod"
    }
  }
}
