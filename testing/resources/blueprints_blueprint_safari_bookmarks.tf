resource "jamfplatform_blueprints_blueprint" "test_safari_bookmarks" {
  name        = "Terraform Test Safari Bookmarks ${var.test_id}"
  description = "Managed by Terraform"

  device_groups = [data.jamfpro_group.test_target_computer_group.group_platform_id]

  safari_bookmarks {
    managed_bookmarks {
      group_identifier = "work-bookmarks"
      title            = "Work Bookmarks"
      bookmarks {
        type  = "bookmark"
        title = "Company Portal"
        url   = "https://portal.company.com"
      }
      bookmarks {
        type  = "bookmark"
        title = "Internal Wiki"
        url   = "https://wiki.company.com"
      }
      bookmarks {
        type  = "folder"
        title = "Development Tools"
        folder {
          title = "GitHub"
          url   = "https://github.com"
        }
        folder {
          title = "Stack Overflow"
          url   = "https://stackoverflow.com"
        }
      }
    }
  }
}
