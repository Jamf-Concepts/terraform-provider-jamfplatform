# Static user group — members are assigned explicitly by Jamf Pro user ID.
resource "jamfplatform_pro_user_group" "exec_team" {
  name       = "Exec Team"
  group_type = "static"

  members = ["12", "34", "56"]
}

# Smart user group — membership is derived from criteria evaluated by Jamf Pro.
# Criteria order is significant: Jamf evaluates left-to-right using the
# supplied `and_or` joins.
resource "jamfplatform_pro_user_group" "managed_apple_ids_vpp_associated" {
  name                        = "Managed Apple IDs with VPP Invitation Associated"
  group_type                  = "smart"
  notify_on_membership_change = true

  criteria = [
    {
      name        = "User Group"
      search_type = "member of"
      value       = "All Managed Apple IDs"
    },
    {
      name        = "VPP Invitation Status"
      search_type = "is"
      value       = "Associated"
      and_or      = "and"
    },
  ]
}
