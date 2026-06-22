# Query for API roles whose display name starts with "Terraform".
list "jamfplatform_pro_api_role" "terraform_roles" {
  provider = jamfplatform

  config {
    filter {
      selector = "displayName"
      argument = "Terraform*"
    }
  }
}
