# Query for buildings in a specific city
list "jamfplatform_pro_building" "buildings_in_minneapolis" {
  provider = jamfplatform

  config {
    filter {
      selector = "city"
      argument = "Minneapolis"
    }
  }
}

# Query for buildings whose name starts with a substring
list "jamfplatform_pro_building" "buildings_by_name_prefix" {
  provider = jamfplatform

  config {
    filter {
      selector = "name"
      argument = "HQ*"
    }
  }
}
