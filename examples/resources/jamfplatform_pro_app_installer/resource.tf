# Reference a category by its .id.
resource "jamfplatform_pro_category" "apps" {
  name     = "Apps"
  priority = 5
}

# Automatic, push-to-all-in-scope App Installer.
# The catalog title is referenced by name (resolved to an ID by the provider);
# list available titles with the jamfplatform_pro_app_installer_titles data source.
resource "jamfplatform_pro_app_installer" "auto" {
  name            = "010 Editor (automatic)"
  app_title_name  = "010 Editor"
  deployment_type = "INSTALL_AUTOMATICALLY"
  update_behavior = "AUTOMATIC"

  category_id    = jamfplatform_pro_category.apps.id
  site_id        = "-1" # -1 means no site
  smart_group_id = "-1" # -1 means no smart group
}

# Self Service deployment with notification and Self Service presentation.
resource "jamfplatform_pro_app_installer" "self_service" {
  name            = "010 Editor (Self Service)"
  app_title_name  = "010 Editor"
  deployment_type = "SELF_SERVICE"
  update_behavior = "AUTOMATIC"

  notification_settings = {
    notification_message = "010 Editor is available to install."
    deadline             = 48
    suppress             = false
  }

  self_service_settings = {
    description                  = "Install 010 Editor from Self Service."
    force_view_description       = true
    include_in_featured_category = true

    categories = [
      {
        category_id = jamfplatform_pro_category.apps.id
        featured    = true
      },
    ]
  }
}
