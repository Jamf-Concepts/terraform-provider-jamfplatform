# Basic SMTP authentication (username + password).
resource "jamfplatform_pro_smtp_server" "this" {
  enabled             = true
  authentication_type = "BASIC"

  sender_settings = {
    email_address = "notifications@example.com"
    display_name  = "Example Notifications"
  }

  connection_settings = {
    host               = "smtp.example.com"
    port               = 587
    encryption_type    = "TLS_1_2" # NONE | SSL | TLS_1_2 | TLS_1_1 | TLS_1 | TLS_1_3
    connection_timeout = 30
  }

  basic_auth_credentials = {
    username = "svc-notifications@example.com"
    # WriteOnly: sent to Jamf Pro but never stored in Terraform state.
    password            = var.smtp_password
    password_wo_version = 1 # bump to rotate the stored password
  }
}

# Microsoft Graph API authentication (no connection_settings).
# resource "jamfplatform_pro_smtp_server" "graph" {
#   authentication_type = "GRAPH_API"
#   sender_settings = { email_address = "notifications@example.com" }
#   graph_api_credentials = {
#     tenant_id                = "00000000-0000-0000-0000-000000000000"
#     client_id                = "11111111-1111-1111-1111-111111111111"
#     client_secret            = var.graph_client_secret
#     client_secret_wo_version = 1
#   }
# }

variable "smtp_password" {
  type      = string
  sensitive = true
}
