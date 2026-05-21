data "jamfplatform_pro_site" "example_by_id" {
  id = "3"
}

data "jamfplatform_pro_site" "example_by_name" {
  name = "Primary"
}

output "site_example_by_id" {
  value = data.jamfplatform_pro_site.example_by_id
}

output "site_example_by_name" {
  value = data.jamfplatform_pro_site.example_by_name
}
