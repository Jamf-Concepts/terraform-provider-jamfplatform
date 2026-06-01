data "jamfplatform_pro_api_role" "example_by_id" {
  id = "42"
}

output "api_role_example_by_id" {
  value = data.jamfplatform_pro_api_role.example_by_id
}
