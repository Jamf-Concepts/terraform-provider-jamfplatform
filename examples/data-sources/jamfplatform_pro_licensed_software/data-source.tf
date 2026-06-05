# Look up a licensed software record by exact name (or set id instead).
data "jamfplatform_pro_licensed_software" "by_name" {
  name = "Acme Editor"
}

# Look up by ID.
data "jamfplatform_pro_licensed_software" "by_id" {
  id = "65"
}
