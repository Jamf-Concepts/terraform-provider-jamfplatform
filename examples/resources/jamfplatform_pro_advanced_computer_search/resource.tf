# Advanced computer search — a saved, criteria-driven inventory query with a
# configurable set of display columns. Criteria order is significant: Jamf
# evaluates left-to-right using the supplied `and_or` joins and parentheses.
resource "jamfplatform_pro_advanced_computer_search" "lab_macs" {
  name = "Lab Macs running Sequoia"

  criteria = [
    {
      name        = "Computer Name"
      search_type = "like"
      value       = "lab"
    },
    {
      name        = "Operating System Version"
      search_type = "greater than or equal"
      value       = "15.0"
      and_or      = "and"
    },
  ]

  # The set of inventory columns shown in the results. Order is not significant
  # — Jamf Pro returns the columns in its own canonical order.
  display_fields = ["Computer Name", "Serial Number", "Last Inventory Update"]

  # Optional: sort the results by up to three display columns.
  sort_1 = "Computer Name"
}

# Minimal search scoped to a site, with no display columns.
resource "jamfplatform_pro_advanced_computer_search" "all_in_site" {
  name    = "All computers in HQ"
  site_id = "1"

  criteria = [
    {
      name        = "Computer Name"
      search_type = "like"
      value       = ""
    },
  ]
}
