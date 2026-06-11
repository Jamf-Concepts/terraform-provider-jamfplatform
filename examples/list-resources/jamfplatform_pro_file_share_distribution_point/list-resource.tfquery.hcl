# List every Jamf Pro file share distribution point.
list "jamfplatform_pro_file_share_distribution_point" "all" {
  provider = jamfplatform
}

# Query for a file share distribution point by name.
list "jamfplatform_pro_file_share_distribution_point" "by_name" {
  provider = jamfplatform

  config {
    filter {
      selector = "name"
      argument = "Main*"
    }
  }
}
