# Minimal webhook with no authentication. Server defaults apply (enabled=true,
# content_type=text/xml, connection_timeout=5, read_timeout=2).
resource "jamfplatform_pro_webhook" "minimal" {
  name  = "Computer added (no auth)"
  url   = "https://example.com/hooks/computer-added"
  event = "ComputerAdded"
}

# Basic authentication. The plaintext password is WriteOnly. Bump
# password_wo_version to rotate it.
resource "jamfplatform_pro_webhook" "basic" {
  name                = "Check-in (basic auth)"
  url                 = "https://example.com/hooks/checkin"
  event               = "ComputerCheckIn"
  content_type        = "application/json"
  authentication_type = "BASIC"
  username            = "webhookuser"
  password            = var.webhook_password # WriteOnly
  password_wo_version = 1
}

# Header authentication. `header` must be a JSON object and is sensitive.
resource "jamfplatform_pro_webhook" "header" {
  name                = "Enrollment (header auth)"
  url                 = "https://example.com/hooks/enroll"
  event               = "MobileDeviceEnrolled"
  authentication_type = "HEADER"
  header              = jsonencode({ Authorization = "Bearer ${var.webhook_token}" })
}

# Hash-signature authentication. `password` is the signing secret (>= 16 chars,
# server-enforced); choose the signature algorithm with hash_algorithm.
resource "jamfplatform_pro_webhook" "hash_signature" {
  name                = "Policy finished (signed)"
  url                 = "https://example.com/hooks/policy"
  event               = "ComputerPolicyFinished"
  authentication_type = "HASH_SIGNATURE"
  password            = var.webhook_signing_secret # WriteOnly, >= 16 chars
  password_wo_version = 1
  hash_algorithm      = "SHA512"
}

# Smart-group membership-change webhook. smart_group_id is only valid for the
# three SmartGroup* events; interpolate a Jamf Pro smart group ID.
resource "jamfplatform_pro_webhook" "smart_group" {
  name                                   = "Smart group membership change"
  url                                    = "https://example.com/hooks/smartgroup"
  event                                  = "SmartGroupComputerMembershipChange"
  smart_group_id                         = 29
  enable_display_fields_for_group_object = true
}

variable "webhook_password" {
  type      = string
  sensitive = true
  default   = null
}

variable "webhook_token" {
  type      = string
  sensitive = true
  default   = null
}

variable "webhook_signing_secret" {
  type      = string
  sensitive = true
  default   = null
}
