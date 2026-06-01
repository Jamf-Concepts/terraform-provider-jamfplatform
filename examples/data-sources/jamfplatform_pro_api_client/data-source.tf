data "jamfplatform_pro_api_client" "example_by_id" {
  id = "88"
}

output "api_client_example_by_id" {
  value = data.jamfplatform_pro_api_client.example_by_id
}
