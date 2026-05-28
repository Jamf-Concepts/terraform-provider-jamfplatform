# Stream every enrollment customization whose display name contains the
# substring "welcome" (case-insensitive). Set `include_resource = true` to
# populate the parent record (description, site, branding palette + icon URL)
# alongside each identity; panes are not included — fetch them per ID via the
# singular resource.

provider "jamfplatform" {}

list "jamfplatform_pro_enrollment_customization" "welcome" {
  provider         = jamfplatform
  include_resource = true

  config {
    filter = {
      name_substring = "welcome"
    }
  }
}
