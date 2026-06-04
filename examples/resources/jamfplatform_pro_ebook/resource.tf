# In-house ebook distributed via Self Service, scoped to all computers.
resource "jamfplatform_pro_ebook" "field_guide" {
  general = {
    name              = "Field Operations Guide"
    author            = "IT Documentation Team"
    url               = "https://files.example.org/field-guide.pdf"
    file_type         = "PDF"
    version           = "2.1"
    deployment_type   = "Make Available in Self Service"
    deploy_as_managed = true
    free              = true
  }

  scope = {
    all_computers      = true
    all_mobile_devices = true
  }

  self_service = {
    display_name             = "Field Operations Guide"
    install_button_text      = "Get"
    self_service_description = "The current field operations handbook."
    feature_on_main_page     = true
  }
}

# App Store ebook — the server derives file_type and version from the Apple
# Books URL, so leave them unset. Scoped to a specific class.
resource "jamfplatform_pro_ebook" "swift_intro" {
  general = {
    name = "Intro to App Development with Swift"
    url  = "https://books.apple.com/us/book/intro-to-app-development-with-swift/id1118575552"
  }

  scope = {
    class_ids = ["12"]
  }
}
