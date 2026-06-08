# List every VPP assignment in the tenant, optionally filtered by a
# case-insensitive name substring. List entries surface as identity-only
# (id and display name); use the data source for full detail.
list "jamfplatform_pro_vpp_assignment" "all" {
  provider = jamfplatform

  config {
    filter = {
      name_substring = "Core"
    }
  }
}
