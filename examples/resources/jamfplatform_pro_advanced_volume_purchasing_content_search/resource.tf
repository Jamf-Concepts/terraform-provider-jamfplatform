# Advanced Volume Purchasing Content search — a saved, criteria-driven query over
# Volume Purchasing (VPP) content with a configurable set of display columns.
#
# NOTE: criteria and display-field names use Jamf Pro's WIRE vocabulary, which
# differs from the admin-UI labels:
#   UI "Content Name"        -> "Name"
#   UI "Price"               -> "Cost"
#   UI "Total Content"       -> "Total"
#   UI "Content Type"        -> "Type"
#   UI "Location"            -> "Account"
#   UI "In Use"              -> "Used"
#   UI "Unassigned Content"  -> "Unused"
#   UI "Volume Assignments"  -> "Assignments"
#   UI "Username"            -> "Username"
resource "jamfplatform_pro_advanced_volume_purchasing_content_search" "office_apps" {
  name = "Office apps with available licenses"

  criteria = [
    {
      name        = "Name"
      search_type = "like"
      value       = "Office"
    },
    {
      name        = "Type"
      search_type = "is"
      value       = "App"
      and_or      = "and"
    },
  ]

  # The set of content columns shown in the results (wire names). Order is not
  # significant — Jamf Pro returns the columns in its own canonical order and
  # silently drops names it does not recognise.
  display_fields = ["Name", "Cost", "Total", "Used"]
}

# Minimal search scoped to a site, with no display columns.
resource "jamfplatform_pro_advanced_volume_purchasing_content_search" "all_in_site" {
  name    = "All content in HQ"
  site_id = "1"

  criteria = [
    {
      name        = "Name"
      search_type = "like"
      value       = ""
    },
  ]
}
