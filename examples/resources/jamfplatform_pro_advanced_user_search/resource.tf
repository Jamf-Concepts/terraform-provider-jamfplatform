# Advanced user search: a saved, criteria-driven user query with a configurable
# set of display columns. Criteria order is significant: Jamf evaluates
# left-to-right using the supplied `and_or` joins and parentheses. Unlike
# advanced computer searches, user searches have no `view_as` or sort columns.
resource "jamfplatform_pro_advanced_user_search" "example_com_users" {
  name = "Users with an example.com email"

  criteria = [
    {
      name        = "Email Address"
      search_type = "like"
      value       = "@example.com"
    },
    {
      name        = "Full Name"
      search_type = "is not"
      value       = "Test User"
      and_or      = "and"
    },
  ]

  # The set of columns shown in the results. Order is not significant; Jamf Pro
  # returns the columns in its own canonical order.
  display_fields = ["Full Name", "Email Address", "Username"]
}
