# Jamf-hosted service discovery ("well-known") settings for Account-Driven enrollment
# (singleton, one record per tenant). Each row sets the Account-Driven enrollment type
# for one synced Apple Business / School Manager (AxM) organization, keyed by the Server
# UUID of its Automated Device Enrollment token. Requires Jamf Pro 11.25.0+.
#
# The set of rows is server-keyed: you can only set enrollment_type on a server_uuid Jamf
# Pro already knows (a synced AxM org). This resource manages only the rows you declare
# (merge) — undeclared orgs are left untouched. To turn off Jamf-hosted service discovery
# for an org, set its enrollment_type = "none"; REMOVING a block stops managing it and
# leaves the current value unchanged.

# Reference the Server UUID from ADE instances managed by this provider:
resource "jamfplatform_pro_service_discovery_enrollment" "this" {
  # Mac computers via Account-Driven Device Enrollment:
  well_known_setting = [
    {
      server_uuid     = jamfplatform_pro_automated_device_enrollment.mac.server_uuid
      enrollment_type = "mdm-adde"
    },
    # iPhone / iPad / Apple Vision Pro via Account-Driven User Enrollment (BYOD):
    {
      server_uuid     = jamfplatform_pro_automated_device_enrollment.mobile.server_uuid
      enrollment_type = "mdm-byod"
    },
  ]
}

# Or set a Server UUID copied from Settings > Automated Device Enrollment > Server UUID
# directly (use the jamfplatform_pro_service_discovery_enrollment data source to discover
# the available Server UUIDs):
# resource "jamfplatform_pro_service_discovery_enrollment" "this" {
#   well_known_setting = [
#     {
#       server_uuid     = "00000000000000000000000000000000"
#       enrollment_type = "mdm-byod"
#     },
#   ]
# }
