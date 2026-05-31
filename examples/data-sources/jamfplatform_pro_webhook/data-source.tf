# Look up a webhook by exact name.
data "jamfplatform_pro_webhook" "by_name" {
  name = "Check-in (basic auth)"
}

# Look up a webhook by ID.
data "jamfplatform_pro_webhook" "by_id" {
  id = "66"
}

output "webhook_url" {
  value = data.jamfplatform_pro_webhook.by_name.url
}
