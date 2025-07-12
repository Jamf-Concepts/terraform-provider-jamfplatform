terraform {
  required_providers {
    jamfplatform = {
      source = "Jamf-Concepts/jamfplatform"
    }
  }
}

provider "jamfplatform" {
  region = "us" # or "eu", "apac"

  cbengine = {
    client_id     = "example-cbengine-client-id"
    client_secret = "example-cbengine-client-secret"
  }
}
