terraform {
  required_providers {
    jamfplatform = {
      source  = "local/jamf/jamfplatform"
      version = "0.1.0"
    }
    jamfpro = {
      source  = "deploymenttheory/jamfpro"
      version = "0.26.0"
    }
  }
}

provider "jamfplatform" {
  base_url      = var.jamfplatform_base_url
  client_id     = var.jamfplatform_client_id
  client_secret = var.jamfplatform_client_secret
}

provider "jamfplatform" {
  alias         = "inventory"
  base_url      = var.jamfplatform_base_url
  client_id     = var.jamfplatform_inventory_client_id
  client_secret = var.jamfplatform_inventory_client_secret
}

provider "jamfpro" {
  auth_method           = "oauth2"
  jamfpro_instance_fqdn = var.jamfpro_instance_fqdn
  client_id             = var.jamfpro_client_id
  client_secret         = var.jamfpro_client_secret
}
