data "jamfplatform_pro_vpp_assignment" "by_id" {
  id = "9"
}

data "jamfplatform_pro_vpp_assignment" "by_name" {
  name = "Volume Purchasing — Core Apps"
}

# Assigned content is surfaced read-only with resolved title names.
output "vpp_assignment_ios_apps" {
  value = data.jamfplatform_pro_vpp_assignment.by_name.ios_apps
}
