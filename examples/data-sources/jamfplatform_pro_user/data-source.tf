data "jamfplatform_pro_user" "example_by_id" {
  id = "42"
}

data "jamfplatform_pro_user" "example_by_username" {
  username = "jsmith"
}

output "user_example_by_id" {
  value = data.jamfplatform_pro_user.example_by_id
}

output "user_example_by_username" {
  value = data.jamfplatform_pro_user.example_by_username
}
