resource "jamfplatform_pro_category" "applications" {
  name     = "Applications"
  priority = 9
}

resource "jamfplatform_pro_category" "security" {
  name     = "Security"
  priority = 5
}
