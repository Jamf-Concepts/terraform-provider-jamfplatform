data "jamfplatform_pro_directory_binding" "example_by_id" {
  id = "1"
}

data "jamfplatform_pro_directory_binding" "example_by_name" {
  name = "ad-prod"
}

output "directory_binding_example_by_id" {
  value = data.jamfplatform_pro_directory_binding.example_by_id
}

output "directory_binding_example_by_name" {
  value = data.jamfplatform_pro_directory_binding.example_by_name
}
