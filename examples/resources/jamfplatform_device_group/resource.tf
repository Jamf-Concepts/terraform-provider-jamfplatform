resource "jamfplatform_device_group" "example_static_mobile_device_group" {
  name        = "Example Static Mobile Device Group"
  description = "An example static mobile device group"
  group_type  = "static"
  device_type = "mobile"
}

resource "jamfplatform_device_group" "example_static_computer_group" {
  name        = "Example Static Computer Group"
  description = "An example static computer group with assigned members"
  group_type  = "static"
  device_type = "computer"
  members     = ["ABCDEF12-3456-7890-ABCD-EF1234567890", "12345678-90AB-CDEF-1234-567890ABCDEF"]
}

resource "jamfplatform_device_group" "example_smart_computer_group" {
  name        = "Example Smart Computer Group"
  group_type  = "smart"
  device_type = "computer"
  description = "An example smart computer group"
  criteria = [
    {
      criteria = "Operating System Version"
      operator = "greater than or equal"
      value    = "26.0"
    },
    {
      and_or                  = "or"
      has_opening_parenthesis = true
      criteria                = "Serial Number"
      operator                = "is"
      value                   = "ABC123456"
    },
    {
      and_or                  = "and"
      criteria                = "Last Enrollment"
      operator                = "before (yyyy-mm-dd)"
      value                   = "2025-01-01"
      has_closing_parenthesis = true
    },
  ]
}

# Directory-service (LDAP / cloud-IdP) group criteria. Write the group by NAME and
# the provider resolves it to the base64 {uuid,serverId} value the API stores
# (and back again on read, so state keeps the readable name). A raw base64 value
# is also accepted verbatim — useful to disambiguate a name that exists on more
# than one directory server, or to paste a value straight from the API/UI.
resource "jamfplatform_device_group" "example_directory_service_group" {
  name        = "Example Directory Service Smart Group"
  group_type  = "smart"
  device_type = "computer"
  description = "Members of a directory-service group"
  criteria = [
    {
      criteria = "Username directory service group"
      operator = "member of"
      value    = "MyDirectoryGroupName" # resolved to base64 on apply
    },
  ]
}

# The Computed `jamf_pro_id` attribute exposes the numeric Jamf Pro classic ID
# for the group. Reference it from classic-API scope blocks (policies,
# configuration profiles, restricted software). The value is null when the API
# integration lacks the Inventory → Device groups → Read permission in Jamf
# Account (API capability `device-groups:read`).
output "smart_computer_group_jamf_pro_id" {
  value = jamfplatform_device_group.example_smart_computer_group.jamf_pro_id
}
