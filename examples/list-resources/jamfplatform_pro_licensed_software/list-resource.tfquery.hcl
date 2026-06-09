# List every licensed software record in the tenant, optionally filtered by a
# case-insensitive name substring. List entries surface as identity-only (id and
# display name); full detail requires a per-record read.
list "jamfplatform_pro_licensed_software" "all" {
  provider = jamfplatform

  config {
    filter = {
      name_substring = "Acme"
    }
  }
}
