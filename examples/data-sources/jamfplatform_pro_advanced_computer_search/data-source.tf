# Look up an advanced computer search by ID.
data "jamfplatform_pro_advanced_computer_search" "by_id" {
  id = "461"
}

# Look up an advanced computer search by exact name.
data "jamfplatform_pro_advanced_computer_search" "by_name" {
  name = "Lab Macs running Sequoia"
}
