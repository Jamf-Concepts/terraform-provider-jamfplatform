variable "jamfplatform_base_url" {
  description = "Jamf Platform base URL for compliance/blueprints (set via TF_VAR_jamfplatform_base_url)"
  type        = string
}

variable "jamfplatform_client_id" {
  description = "OAuth client ID for compliance/blueprints (set via TF_VAR_jamfplatform_client_id)"
  type        = string
  sensitive   = true
}

variable "jamfplatform_client_secret" {
  description = "OAuth client secret for compliance/blueprints (set via TF_VAR_jamfplatform_client_secret)"
  type        = string
  sensitive   = true
}

variable "jamfplatform_inventory_client_id" {
  description = "OAuth client ID for inventory APIs (set via TF_VAR_jamfplatform_inventory_client_id, defaults to null)"
  type        = string
  default     = null
  sensitive   = true
}

variable "jamfplatform_inventory_client_secret" {
  description = "OAuth client secret for inventory APIs (set via TF_VAR_jamfplatform_inventory_client_secret, defaults to null)"
  type        = string
  sensitive   = true
}

variable "jamfpro_instance_fqdn" {
  description = "Jamf Pro instance FQDN (set via TF_VAR_jamfpro_instance_fqdn)"
  type        = string
}

variable "jamfpro_client_id" {
  description = "OAuth client ID for Jamf Pro (set via TF_VAR_jamfpro_client_id)"
  type        = string
  sensitive   = true
}

variable "jamfpro_client_secret" {
  description = "OAuth client secret for Jamf Pro (set via TF_VAR_jamfpro_client_secret)"
  type        = string
  sensitive   = true
}

variable "test_id" {
  description = "Unique identifier for test resources to avoid naming conflicts"
  type        = string
  default     = "tf-test"
}
