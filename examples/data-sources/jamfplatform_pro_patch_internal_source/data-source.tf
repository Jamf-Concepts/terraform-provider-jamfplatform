# The built-in "Jamf" internal source is present on every tenant. Looking it up
# also returns its available_titles catalog, which you can use to discover the
# name_id for a jamfplatform_pro_patch_software_title.
data "jamfplatform_pro_patch_internal_source" "jamf" {
  name = "Jamf"
}

# Look up by ID instead.
data "jamfplatform_pro_patch_internal_source" "by_id" {
  id = "1"
}

output "patch_internal_source_jamf" {
  value = data.jamfplatform_pro_patch_internal_source.jamf
}

# Project the catalog down to a name -> name_id map for easy reference.
output "jamf_title_name_ids" {
  value = {
    for t in data.jamfplatform_pro_patch_internal_source.jamf.available_titles :
    t.app_name => t.name_id
  }
}
