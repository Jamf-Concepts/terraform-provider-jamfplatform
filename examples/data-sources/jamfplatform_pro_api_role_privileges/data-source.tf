# Read the full set of valid Jamf Pro privilege strings for this tenant.
data "jamfplatform_pro_api_role_privileges" "all" {}

# Narrow to privileges whose name contains "API".
data "jamfplatform_pro_api_role_privileges" "api" {
  search = "API"
}

output "api_privileges" {
  value = data.jamfplatform_pro_api_role_privileges.api.privileges
}
