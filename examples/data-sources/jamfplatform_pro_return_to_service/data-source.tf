# Look up a Return to Service configuration by ID.
data "jamfplatform_pro_return_to_service" "by_id" {
  id = "1"
}

# Look up a Return to Service configuration by exact display name. Display names
# are not guaranteed unique — a name matching more than one configuration
# returns an error.
data "jamfplatform_pro_return_to_service" "by_name" {
  display_name = "Front Desk iPads"
}
