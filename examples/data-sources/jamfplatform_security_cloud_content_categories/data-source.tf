# The Jamf-curated content category catalogue. Read-only, and the same for every
# entitled tenant.
data "jamfplatform_security_cloud_content_categories" "all" {}

# A category carries two names. "display_name" is the one to reference; "name" is
# Jamf's internal label ("Category - Social") and is informational only.
locals {
  category_ids_by_display_name = {
    for category in data.jamfplatform_security_cloud_content_categories.all.content_categories :
    category.display_name => category.id
  }

  social = local.category_ids_by_display_name["Social"]
}

output "social_category_id" {
  value = local.social
}
