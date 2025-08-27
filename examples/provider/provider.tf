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
}
