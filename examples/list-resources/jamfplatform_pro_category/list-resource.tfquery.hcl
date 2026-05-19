# Query for high-priority Jamf Pro categories
list "jamfplatform_pro_category" "high_priority_categories" {
  provider = jamfplatform

  config {
    filter {
      selector = "priority"
      operator = "<="
      argument = "5"
    }
  }
}

# Query for categories whose name starts with a substring
list "jamfplatform_pro_category" "categories_by_name_prefix" {
  provider = jamfplatform

  config {
    filter {
      selector = "name"
      argument = "App*"
    }
  }
}
