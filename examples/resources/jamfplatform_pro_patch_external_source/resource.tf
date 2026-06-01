resource "jamfplatform_pro_patch_external_source" "datajar" {
  name                           = "Jamf Auto Update"
  enabled                        = true
  host_name                      = "definitions.datajar.mobi/v2/"
  ssl_enabled                    = true
  certificate_validation_enabled = false
}

resource "jamfplatform_pro_patch_external_source" "custom" {
  name      = "Internal Definitions"
  host_name = "patch.internal.example.com"
  port      = 8443
}
