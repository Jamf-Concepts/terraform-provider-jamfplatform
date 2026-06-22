# Look up a Jamf Pro package by its ID.
data "jamfplatform_pro_package" "by_id" {
  id = "123"
}

# Look up a Jamf Pro package by its display name (exact match).
data "jamfplatform_pro_package" "by_name" {
  display_name = "MyApp 1.0.0"
}
