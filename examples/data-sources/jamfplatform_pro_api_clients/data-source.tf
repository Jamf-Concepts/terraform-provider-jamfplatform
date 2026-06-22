# Search API clients whose display name starts with "Terraform".
data "jamfplatform_pro_api_clients" "terraform_clients" {
  filter = [
    {
      selector = "displayName"
      argument = "Terraform*"
    }
  ]
}

output "terraform_clients" {
  value = data.jamfplatform_pro_api_clients.terraform_clients.api_clients
}
