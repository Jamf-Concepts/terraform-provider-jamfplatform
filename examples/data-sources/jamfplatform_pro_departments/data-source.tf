data "jamfplatform_pro_departments" "engineering" {
  filter = [
    {
      selector = "name"
      argument = "Engineering"
    }
  ]
}

data "jamfplatform_pro_departments" "by_name_prefix" {
  filter = [
    {
      selector = "name"
      argument = "Eng*"
    }
  ]
}

output "engineering_departments" {
  value = data.jamfplatform_pro_departments.engineering.departments
}
