# Discover the privilege strings grantable on this tenant, to look up exact
# values for an account / account_group `privileges` block.
data "jamfplatform_pro_account_privileges" "catalog" {}

output "all_grantable_privileges" {
  value = data.jamfplatform_pro_account_privileges.catalog.all
}

output "computer_object_privileges" {
  value = [
    for p in data.jamfplatform_pro_account_privileges.catalog.jamf_pro_server_objects : p
    if can(regex("Computers", p))
  ]
}
