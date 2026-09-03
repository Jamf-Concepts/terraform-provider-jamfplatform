# A device group is how Jamf Security Cloud assigns access: an app's assignments
# name the groups whose members may reach it, and UEM Connect maps the Jamf Pro
# groups it syncs onto them.
#
# The group itself holds nothing but a name. Membership is decided by whatever
# references the group, never on the group.

resource "jamfplatform_security_cloud_device_group" "executives" {
  name = "Executives"
}

resource "jamfplatform_security_cloud_device_group" "field_staff" {
  name = "Field Staff"
}

# Names must be unique on the tenant, and Jamf Security Cloud compares them
# exactly. "Contractors" and "contractors" are two different groups, which is
# rarely what anyone wants. Leading and trailing whitespace is rejected at plan
# time, because Jamf Security Cloud would silently strip it and Terraform would
# then report a result it did not ask for.
resource "jamfplatform_security_cloud_device_group" "contractors" {
  name = "Contractors"
}

# Removing a group does not fail when something still points at it: Jamf Security
# Cloud deletes the group and quietly drops it from every assignment and mapping
# that named it, which can leave those objects assigned to nobody. Referencing the
# group by ID rather than by a copied-out string at least means Terraform knows
# about the dependency and destroys in the right order.
output "executives_group_id" {
  value = jamfplatform_security_cloud_device_group.executives.id
}
