# Read the current Self Service iOS & iPadOS branding configuration.
data "jamfplatform_pro_self_service_branding_ios" "current" {}

output "ios_main_header" {
  value = data.jamfplatform_pro_self_service_branding_ios.current.main_header
}
