# Read the current Jamf Pro SMTP Server settings. WriteOnly secrets are never returned.
data "jamfplatform_pro_smtp_server" "current" {}

output "smtp_authentication_type" {
  value = data.jamfplatform_pro_smtp_server.current.authentication_type
}
