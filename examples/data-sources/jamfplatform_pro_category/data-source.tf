data "jamfplatform_pro_category" "example_by_id" {
  id = "5"
}

output "category_example_by_id" {
  value = data.jamfplatform_pro_category.example_by_id
}
