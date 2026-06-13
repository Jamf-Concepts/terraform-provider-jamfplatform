data "jamfplatform_pro_allowed_file_extension" "example_by_id" {
  id = "7"
}

data "jamfplatform_pro_allowed_file_extension" "example_by_extension" {
  extension = "csv"
}

output "allowed_file_extension_example_by_id" {
  value = data.jamfplatform_pro_allowed_file_extension.example_by_id
}

output "allowed_file_extension_example_by_extension" {
  value = data.jamfplatform_pro_allowed_file_extension.example_by_extension
}
