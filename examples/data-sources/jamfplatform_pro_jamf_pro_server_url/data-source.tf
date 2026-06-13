# Read the Jamf Pro server URL that devices check in against.
data "jamfplatform_pro_jamf_pro_server_url" "current" {}

output "jamf_pro_server_url" {
  value = data.jamfplatform_pro_jamf_pro_server_url.current.url
}
