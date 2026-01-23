terraform {
  required_providers {
    jamfplatform = {
      source  = "local/jamf/jamfplatform"
      version = "0.1.0"
    }
  }
}

provider "jamfplatform" {
  base_url      = var.jamfplatform_base_url
  client_id     = var.jamfplatform_client_id
  client_secret = var.jamfplatform_client_secret
}
