terraform {
  required_providers {
    jamfplatform = {
      source = "Jamf-Concepts/jamfplatform"
    }
  }
}

provider "jamfplatform" {
  region        = "us" # or "eu", "apac"
  client_id     = "example-client-id"
  client_secret = "example-client-secret"
}
