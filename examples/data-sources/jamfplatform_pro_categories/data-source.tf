data "jamfplatform_pro_categories" "high_priority" {
  filter = [
    {
      selector = "priority"
      operator = "<="
      argument = "5"
    }
  ]
}

data "jamfplatform_pro_categories" "by_name_prefix" {
  filter = [
    {
      selector = "name"
      argument = "App*"
    }
  ]
}

output "high_priority_categories" {
  value = data.jamfplatform_pro_categories.high_priority.categories
}
