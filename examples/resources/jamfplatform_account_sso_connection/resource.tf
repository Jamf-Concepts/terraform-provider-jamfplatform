# An OIDC single sign-on connection for one or more verified domains. Users with
# an email address at a listed domain are redirected to this identity provider
# instead of signing in with a Jamf ID.
#
# Jamf refuses a connection naming a domain that is not yet verified, and
# verification cannot happen in the same run as the claim — Jamf allows one check
# every five minutes per domain and claiming it starts that clock. So this takes
# three applies, not one:
#
#   1. apply the domain below, then publish its verification_txt_record in DNS
#   2. terraform apply -invoke='action.jamfplatform_account_sso_domain_verify.corp'
#   3. apply again with the connection added
#
# See the jamfplatform_account_sso_domain_verify example for that action.
resource "jamfplatform_account_sso_domain" "corp" {
  domain = "example.com"
}

# Any change to a connection replaces it: Jamf Account has no working update
# endpoint today, so Terraform cannot edit one in place.
#
# That matters for a connection carrying real sign-in traffic. Terraform destroys
# before it creates by default, and while the connection is gone nobody on its
# domains can authenticate through it. create_before_destroy closes that gap —
# Jamf allows two connections on the same domain, so the replacement can exist
# before the original is removed.
resource "jamfplatform_account_sso_connection" "corp" {
  name            = "Corp OIDC"
  connection_type = "generic_oidc"
  hosting_region  = "US"

  client_id     = var.idp_client_id
  client_secret = var.idp_client_secret

  # Bump this to rotate the secret. The value itself is never readable back, so
  # a change to client_secret alone cannot be detected.
  client_secret_wo_version = 1

  scopes = "openid email profile groups"

  generic_oidc = {
    issuer_url             = "https://idp.example.com"
    authorization_endpoint = "https://idp.example.com/authorize"
    token_endpoint         = "https://idp.example.com/token"
    jwks_uri               = "https://idp.example.com/.well-known/jwks.json"
  }

  domains = [jamfplatform_account_sso_domain.corp.domain]

  # Only groups whose name contains one of these reach Jamf. An empty groups
  # list with an operator set means no filtering, which is not the same as
  # omitting the block.
  group_name_filter = {
    operator = "or"
    groups   = ["jamf-admins", "jamf-users"]
  }

  # Blank inherits the organization default. Maximum is 1440 minutes.
  session_duration_minutes   = 480
  inactivity_timeout_minutes = 60

  # Which Jamf products this connection signs users in to. Jamf does not report
  # the tenant list back, so a change made in the Jamf Account console is
  # invisible here — treat this configuration as the source of truth.
  enabled_products = [
    {
      product = "ACCOUNT"
      tenants = []
    },
  ]

  lifecycle {
    # Build the replacement before removing the connection in use, so editing one
    # does not interrupt sign-in.
    create_before_destroy = true
  }
}
variable "idp_client_id" {
  type        = string
  description = "Client ID of the application registered with your identity provider."
}

variable "idp_client_secret" {
  type        = string
  sensitive   = true
  description = "Client secret of that application."
}
