resource "jamfplatform_blueprints_blueprint" "test_safari_bookmarks" {
  name        = "Terraform Test Safari Bookmarks ${var.test_id}"
  description = "Managed by Terraform"
  deployed    = false

  device_groups = [jamfplatform_device_group.test_smart_computer.id]

  safari_bookmarks = {
    managed_bookmarks = [{
      group_identifier = "work-bookmarks"
      title            = "Work Bookmarks"
      bookmarks = [{
        type  = "bookmark"
        title = "Company Portal"
        url   = "https://portal.company.com"
      }, {
        type  = "bookmark"
        title = "Internal Wiki"
        url   = "https://wiki.company.com"
      }, {
        type  = "folder"
        title = "Development Tools"
        folder = [{
          title = "GitHub"
          url   = "https://github.com"
        }, {
          title = "Stack Overflow"
          url   = "https://stackoverflow.com"
        }]
      }]
    }]
  }
}
