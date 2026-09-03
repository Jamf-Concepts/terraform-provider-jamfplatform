# List every App Installer deployment in the tenant, optionally filtered by a
# case-insensitive name substring. List entries surface as identity-only
# (id and display name); use the data sources for full detail.
list "jamfplatform_pro_app_installer" "all" {
  provider = jamfplatform

  config {
    filter = {
      name_substring = "Chrome"
    }
  }
}
