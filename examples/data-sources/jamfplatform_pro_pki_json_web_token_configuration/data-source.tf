# Look up a JSON Web Token configuration by exact name.
data "jamfplatform_pro_pki_json_web_token_configuration" "by_name" {
  name = "Jamf Setup token"
}

# Look up a JSON Web Token configuration by ID.
data "jamfplatform_pro_pki_json_web_token_configuration" "by_id" {
  id = "33"
}

output "token_expiry" {
  value = data.jamfplatform_pro_pki_json_web_token_configuration.by_name.token_expiry
}
