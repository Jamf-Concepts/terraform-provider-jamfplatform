resource "jamfplatform_pro_api_role" "automation" {
  display_name = "Terraform Automation"
  privileges = [
    "Read Computers",
    "Update Computers",
  ]
}

resource "jamfplatform_pro_api_client" "automation" {
  display_name                  = "Terraform Automation Client"
  api_roles                     = [jamfplatform_pro_api_role.automation.display_name]
  enabled                       = true
  access_token_lifetime_seconds = 300

  # Set credential_rotation to mint the OAuth client secret (the client must be
  # enabled). Change the value to rotate the secret. The generated client_secret
  # is stored — sensitive — in Terraform state, as Jamf Pro never returns it again.
  credential_rotation = "1"
}

output "client_id" {
  value = jamfplatform_pro_api_client.automation.client_id
}

output "client_secret" {
  value     = jamfplatform_pro_api_client.automation.client_secret
  sensitive = true
}
