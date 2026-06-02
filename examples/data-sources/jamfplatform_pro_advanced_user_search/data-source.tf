# Look up an advanced user search by ID.
data "jamfplatform_pro_advanced_user_search" "by_id" {
  id = "463"
}

# Look up an advanced user search by exact name.
data "jamfplatform_pro_advanced_user_search" "by_name" {
  name = "Users with an example.com email"
}
