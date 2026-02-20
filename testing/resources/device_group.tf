resource "jamfplatform_device_group" "test_static_computer" {
  name        = "Terraform Test Static Group ${var.test_id}"
  group_type  = "static"
  device_type = "computer"
  description = "Managed by Terraform"
}

resource "jamfplatform_device_group" "test_smart_computer" {
  name        = "Terraform Test Smart Group ${var.test_id}"
  group_type  = "smart"
  device_type = "computer"
  description = "Managed by Terraform"

  criteria = [
    {
      criteria = "Operating System Version"
      operator = "is"
      value    = "13.4.1"
    },
    {
      and_or                  = "and"
      criteria                = "Serial Number"
      operator                = "is not"
      value                   = ""
      has_opening_parenthesis = true
    },
    {
      and_or                  = "or"
      criteria                = "Last Enrollment"
      operator                = "before (yyyy-mm-dd)"
      value                   = "2025-01-01"
      has_closing_parenthesis = true
    },
  ]
}

resource "jamfplatform_device_group" "test_static_mobile" {
  name        = "Terraform Test Static Group ${var.test_id}"
  group_type  = "static"
  device_type = "mobile"
  description = "Managed by Terraform"
}

resource "jamfplatform_device_group" "test_smart_mobile" {
  name        = "Terraform Test Smart Group ${var.test_id}"
  group_type  = "smart"
  device_type = "mobile"
  description = "Managed by Terraform"

  criteria = [
    {
      criteria = "OS Version"
      operator = "is"
      value    = "13.4.1"
    },
    {
      and_or                  = "and"
      criteria                = "Serial Number"
      operator                = "is not"
      value                   = ""
      has_opening_parenthesis = true
    },
    {
      and_or                  = "or"
      criteria                = "Last Enrollment"
      operator                = "before (yyyy-mm-dd)"
      value                   = "2025-01-01"
      has_closing_parenthesis = true
    },
  ]
}
