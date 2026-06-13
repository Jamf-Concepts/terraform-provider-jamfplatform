# Read the valid App Store country/region codes for the tenant — the values
# accepted by jamfplatform_pro_app_request_settings.app_store_locale (alongside
# the literal "deviceLocale").

data "jamfplatform_pro_app_store_country_codes" "all" {}

# Narrow the list with a case-insensitive substring (matched on code and name).
data "jamfplatform_pro_app_store_country_codes" "united" {
  search = "united"
}

output "app_store_country_codes" {
  value = data.jamfplatform_pro_app_store_country_codes.all.country_codes
}
