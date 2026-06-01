data "jamfplatform_pro_patch_external_source" "example_by_id" {
  id = "3"
}

data "jamfplatform_pro_patch_external_source" "example_by_name" {
  name = "Jamf Auto Update"
}

output "patch_external_source_example_by_id" {
  value = data.jamfplatform_pro_patch_external_source.example_by_id
}

output "patch_external_source_example_by_name" {
  value = data.jamfplatform_pro_patch_external_source.example_by_name
}
