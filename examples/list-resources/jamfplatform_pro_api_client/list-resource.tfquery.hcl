# Query for API clients whose display name starts with "Terraform".
list "jamfplatform_pro_api_client" "terraform_clients" {
  provider = jamfplatform

  config {
    filter {
      selector = "displayName"
      argument = "Terraform*"
    }
  }
}
