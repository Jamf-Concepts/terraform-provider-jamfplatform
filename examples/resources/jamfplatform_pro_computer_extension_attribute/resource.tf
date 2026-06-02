# Text field extension attribute shown under the General inventory tab.
resource "jamfplatform_pro_computer_extension_attribute" "asset_tag" {
  name              = "Asset Tag"
  description       = "Physical asset tag entered by a technician."
  data_type         = "STRING"
  input_type        = "TEXT"
  inventory_display = "GENERAL"
}

# Pop-up menu extension attribute with a fixed, ordered list of choices.
resource "jamfplatform_pro_computer_extension_attribute" "department_class" {
  name               = "Department Class"
  data_type          = "STRING"
  input_type         = "POPUP"
  inventory_display  = "USER_AND_LOCATION"
  popup_menu_choices = ["Standard", "Restricted", "Kiosk"]
}

# Script extension attribute. Only SCRIPT extension attributes may be disabled,
# and `script` is required. `manage_existing_data` is a write-only flag.
resource "jamfplatform_pro_computer_extension_attribute" "quarantine_status" {
  name                 = "Quarantine Status"
  data_type            = "STRING"
  input_type           = "SCRIPT"
  inventory_display    = "GENERAL"
  enabled              = true
  manage_existing_data = "RETAIN"
  script               = <<-EOT
    #!/bin/bash
    if [[ -d "/Library/Application Support/JamfProtect/Quarantine" ]]; then
      echo "<result>Present</result>"
    else
      echo "<result>Absent</result>"
    fi
  EOT
}

# Directory-service-mapped extension attribute. `directory_service_attribute` is
# required; set `allow_multiple_values` to collect more than one value.
resource "jamfplatform_pro_computer_extension_attribute" "ad_department" {
  name                        = "AD Department"
  data_type                   = "STRING"
  input_type                  = "DIRECTORY_SERVICE_ATTRIBUTE_MAPPING"
  inventory_display           = "USER_AND_LOCATION"
  directory_service_attribute = "department"
  allow_multiple_values       = false
}
