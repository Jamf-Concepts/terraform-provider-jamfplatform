# Advanced mobile device search: a saved, criteria-driven inventory query with a
# configurable set of display columns. Criteria order is significant: Jamf
# evaluates left-to-right using the supplied `and_or` joins and parentheses.
resource "jamfplatform_pro_advanced_mobile_device_search" "unmanaged_ipads" {
  name = "Unmanaged supervised iPads"

  criteria = [
    {
      name        = "Managed"
      search_type = "is"
      value       = "Unmanaged"
    },
    {
      name        = "Supervised"
      search_type = "is"
      value       = "Supervised"
      and_or      = "and"
    },
  ]

  # The set of inventory columns shown in the results. Order is not
  # significant; Jamf Pro returns the columns in its own canonical order.
  display_fields = ["Display Name", "Serial Number", "Last Inventory Update"]
}

# Minimal search scoped to a site, with no display columns.
resource "jamfplatform_pro_advanced_mobile_device_search" "all_in_site" {
  name    = "All devices in HQ"
  site_id = "1"

  criteria = [
    {
      name        = "Display Name"
      search_type = "like"
      value       = ""
    },
  ]
}
