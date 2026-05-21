# Query for a department by exact name
list "jamfplatform_pro_department" "engineering" {
  provider = jamfplatform

  config {
    filter {
      selector = "name"
      argument = "Engineering"
    }
  }
}

# Query for departments whose name starts with a substring
list "jamfplatform_pro_department" "departments_by_name_prefix" {
  provider = jamfplatform

  config {
    filter {
      selector = "name"
      argument = "Eng*"
    }
  }
}
