terraform {
  required_providers {
    jamfplatform = {
      source = "Jamf-Concepts/jamfplatform"
    }
  }
}

provider "jamfplatform" {
  base_url      = "https://us.apigw.jamf.com" # or "https://eu.apigw.jamf.com", "https://apac.apigw.jamf.com"
  client_id     = "example-client-id"
  client_secret = "example-client-secret"
  # The "Platform environment" your API integration targets. This is the
  # preferred scope for new integrations.
  environment_id = "00000000-0000-0000-0000-000000000000"

  # Legacy alternative: the single "Tenant" your API integration targets. Set
  # exactly one of environment_id or tenant_id, and set the one your integration
  # was actually created for.
  # tenant_id = "00000000-0000-0000-0000-000000000000"
}
