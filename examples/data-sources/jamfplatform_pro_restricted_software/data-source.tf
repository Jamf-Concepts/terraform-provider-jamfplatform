# Look up a restricted software record by exact name.
data "jamfplatform_pro_restricted_software" "by_name" {
  name = "Block Chess"
}

output "block_chess_id" {
  value = data.jamfplatform_pro_restricted_software.by_name.id
}

# Look up a restricted software record by ID.
data "jamfplatform_pro_restricted_software" "by_id" {
  id = "10"
}
