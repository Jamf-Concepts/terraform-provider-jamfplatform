# Text field mobile device extension attribute. Mobile EAs cannot run scripts.
resource "jamfplatform_pro_mobile_device_extension_attribute" "warranty_ref" {
  name              = "Warranty Reference"
  description       = "Vendor warranty reference number."
  data_type         = "STRING"
  input_type        = "TEXT"
  inventory_display = "GENERAL"
}

# Pop-up menu mobile device extension attribute.
resource "jamfplatform_pro_mobile_device_extension_attribute" "device_role" {
  name               = "Device Role"
  data_type          = "STRING"
  input_type         = "POPUP"
  inventory_display  = "EXTENSION_ATTRIBUTES"
  popup_menu_choices = ["Shared", "Assigned", "Loaner"]
}

# Directory-service-mapped mobile device extension attribute.
resource "jamfplatform_pro_mobile_device_extension_attribute" "ad_office" {
  name                        = "AD Office"
  data_type                   = "STRING"
  input_type                  = "DIRECTORY_SERVICE_ATTRIBUTE_MAPPING"
  inventory_display           = "USER_AND_LOCATION"
  directory_service_attribute = "physicalDeliveryOfficeName"
}
