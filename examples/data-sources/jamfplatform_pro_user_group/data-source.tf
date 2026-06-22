data "jamfplatform_pro_user_group" "example_by_id" {
  id = "3"
}

data "jamfplatform_pro_user_group" "example_by_name" {
  name = "Exec Team"
}

output "user_group_example_by_id" {
  value = data.jamfplatform_pro_user_group.example_by_id
}

output "user_group_example_by_name" {
  value = data.jamfplatform_pro_user_group.example_by_name
}
