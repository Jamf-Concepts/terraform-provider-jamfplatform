# Minimal App Store Mac app. name, version, and bundle_id are required and
# stored verbatim. There is no App Store metadata resolution.
resource "jamfplatform_pro_mac_app_store_app" "minimal" {
  general = {
    name      = "iMovie"
    version   = "10.4.3"
    bundle_id = "com.apple.iMovieApp"
    url       = "https://apps.apple.com/app/imovie/id408981434"
  }
}

# Self Service distribution scoped to every computer, with a Self Service
# listing. deployment_type defaults to "Make Available in Self Service".
resource "jamfplatform_pro_mac_app_store_app" "self_service" {
  general = {
    name            = "Keynote"
    version         = "14.1"
    bundle_id       = "com.apple.iWork.Keynote"
    url             = "https://apps.apple.com/app/keynote/id409183694"
    deployment_type = "Make Available in Self Service"
    category_id     = jamfplatform_pro_category.productivity.id
  }

  scope = {
    targets = {
      all_computers = true
    }
  }

  self_service = {
    install_button_text             = "Install"
    self_service_description        = "Apple Keynote — managed by Terraform."
    feature_on_main_page            = true
    force_users_to_view_description = false

    self_service_categories = [
      {
        id         = jamfplatform_pro_category.productivity.id
        display_in = true
        feature_in = false
      },
    ]
  }
}

# Automatic install scoped to a computer group, sourced from a Platform
# Services device group via .jamf_pro_id.
resource "jamfplatform_pro_mac_app_store_app" "automatic" {
  general = {
    name            = "Numbers"
    version         = "14.1"
    bundle_id       = "com.apple.iWork.Numbers"
    url             = "https://apps.apple.com/app/numbers/id409203825"
    deployment_type = "Install Automatically/Prompt Users to Install"
  }

  scope = {
    targets = {
      computer_group_ids = [
        jamfplatform_device_group.engineering.jamf_pro_id,
      ]
    }

    exclusions = {
      department_ids = [jamfplatform_pro_department.it.id]
    }
  }
}
