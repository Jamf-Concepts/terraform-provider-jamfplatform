# Read the current Self Service macOS branding configuration.
data "jamfplatform_pro_self_service_branding_macos" "current" {}

output "macos_application_header" {
  value = data.jamfplatform_pro_self_service_branding_macos.current.application_header
}
