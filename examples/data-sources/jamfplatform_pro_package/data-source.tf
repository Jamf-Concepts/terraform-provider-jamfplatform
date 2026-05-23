# Look up a Jamf Pro package by its server-assigned ID.
data "jamfplatform_pro_package" "by_id" {
  id = "123"
}

# Look up a Jamf Pro package by its display name (wire field `packageName`,
# exact match). Resolves via the V1 list endpoint with an RSQL
# packageName=="..." clause server-side.
data "jamfplatform_pro_package" "by_name" {
  display_name = "MyApp 1.0.0"
}
