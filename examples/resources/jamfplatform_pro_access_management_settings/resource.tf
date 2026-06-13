# Access Management settings for Managed Apple Accounts (singleton, one per tenant).
# Names the Automated Device Enrollment (ADE) server object Jamf Pro returns in its
# Get Token response so Apple Business / School Manager can restrict Managed Apple
# Account sign-in to managed or supervised devices. Requires Jamf Pro 11.18.0+.

# Reference the Server UUID from an ADE instance managed by this provider:
resource "jamfplatform_pro_access_management_settings" "this" {
  automated_device_enrollment_server_uuid = jamfplatform_pro_automated_device_enrollment.example.server_uuid
}

# Or set a Server UUID copied from Settings > Automated Device Enrollment directly:
# resource "jamfplatform_pro_access_management_settings" "this" {
#   automated_device_enrollment_server_uuid = "00000000-0000-0000-0000-000000000000"
# }

# To clear the setting (no ADE server configured), set it to an empty string.
# Omitting the attribute preserves whatever is currently set on the tenant.
