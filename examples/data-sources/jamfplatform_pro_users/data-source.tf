# All inventory users.
data "jamfplatform_pro_users" "all" {}

# Users in the Jamf domain whose position contains "Manager".
data "jamfplatform_pro_users" "jamf_managers" {
  filter = [
    {
      selector = "email"
      argument = "*@jamf.com"
    },
    {
      selector = "position"
      operator = "=="
      argument = "*Manager*"
    }
  ]
}

output "all_users" {
  value = data.jamfplatform_pro_users.all.users
}

output "jamf_managers" {
  value = data.jamfplatform_pro_users.jamf_managers.users
}
