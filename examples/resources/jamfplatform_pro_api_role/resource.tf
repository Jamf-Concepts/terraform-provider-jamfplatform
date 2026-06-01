resource "jamfplatform_pro_api_role" "read_only" {
  display_name = "Terraform Read-Only"
  privileges = [
    "Read Computers",
    "Read Mobile Devices",
    "Read API Roles",
    "Read API Integrations",
  ]
}

# Discover the valid privilege strings for this tenant / Jamf Pro version.
data "jamfplatform_pro_api_role_privileges" "all" {}

output "all_privileges" {
  value = data.jamfplatform_pro_api_role_privileges.all.privileges
}
