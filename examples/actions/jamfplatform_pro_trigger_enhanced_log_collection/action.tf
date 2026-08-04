# Starts an AppleCare enhanced log collection session so diagnostic logs are
# collected and uploaded to Apple for an AppleCare escalation.
#
# Requires Jamf Pro 11.30 or later, and a device running iOS, iPadOS, tvOS or
# macOS 27.0 or later. Jamf Pro queues the command for any targeted device, but a
# device on an earlier OS cannot act on it.
action "jamfplatform_pro_trigger_enhanced_log_collection" "applecare_escalation" {
  config {
    # The AppleCare ticket normally concerns one device, so target one device.
    serial_numbers = ["C02XXXXXXXXX"]

    # Supplied by AppleCare as part of the ticket. Keep it in a variable or a
    # secret store rather than committing it: action attributes cannot be marked
    # sensitive, so this value appears in Terraform plan output.
    apple_care_token = var.applecare_token
  }
}

variable "applecare_token" {
  description = "AppleCare token authorising the enhanced log collection session."
  type        = string
}

# The same token is sent to every device in the batch. Only batch when AppleCare
# issued a token valid for more than one device — Apple does not document whether
# a token is device-scoped, and devices that reject it fail individually in their
# own management history rather than failing the invocation.
action "jamfplatform_pro_trigger_enhanced_log_collection" "fleet_escalation" {
  config {
    management_ids = [
      "11111111-1111-1111-1111-111111111111",
      "22222222-2222-2222-2222-222222222222",
    ]
    apple_care_token = var.applecare_token
  }
}
