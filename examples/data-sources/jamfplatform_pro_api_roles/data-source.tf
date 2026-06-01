# Search API roles whose display name contains "Read".
data "jamfplatform_pro_api_roles" "read_roles" {
  filter = [
    {
      selector = "displayName"
      argument = "*Read*"
    }
  ]
}

output "read_roles" {
  value = data.jamfplatform_pro_api_roles.read_roles.api_roles
}
