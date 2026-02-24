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
