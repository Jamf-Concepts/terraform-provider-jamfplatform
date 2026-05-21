data "jamfplatform_pro_department" "example_by_id" {
  id = "12"
}

output "department_example_by_id" {
  value = data.jamfplatform_pro_department.example_by_id
}
