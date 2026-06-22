# Minimal App Store app. name, version, and bundle_id are always required.
# An itunes_store_url marks it as an App Store title, so os_type is not needed.
# (os_type is only required for in-house apps, which have no itunes_store_url.)
resource "jamfplatform_pro_mobile_device_app" "minimal" {
  general = {
    name             = "Maps"
    version          = "1.0"
    bundle_id        = "com.apple.Maps"
    itunes_store_url = "https://apps.apple.com/us/app/apple-maps/id915056765"
  }
}

# Self Service distribution scoped to every mobile device, with a Self Service
# listing. deployment_type defaults to "Make Available in Self Service".
resource "jamfplatform_pro_mobile_device_app" "self_service" {
  general = {
    name             = "Pages"
    version          = "14.1"
    bundle_id        = "com.apple.Pages"
    itunes_store_url = "https://apps.apple.com/us/app/pages/id361309726"
    deployment_type  = "Make Available in Self Service"
    category_id      = jamfplatform_pro_category.productivity.id
  }

  scope = {
    targets = {
      all_mobile_devices = true
    }
  }

  self_service = {
    install_button_text       = "Install"
    after_install_button_text = "Open"
    self_service_description  = "Apple Pages — managed by Terraform."
    feature_on_main_page      = true
    notification_enabled      = true

    self_service_categories = [
      {
        id         = jamfplatform_pro_category.productivity.id
        display_in = true
      },
    ]
  }
}

# Automatic install scoped to a mobile device group (sourced from a Platform
# Services device group via .jamf_pro_id), with managed-app configuration.
resource "jamfplatform_pro_mobile_device_app" "automatic" {
  general = {
    name                  = "Numbers"
    version               = "14.1"
    bundle_id             = "com.apple.Numbers"
    itunes_store_url      = "https://apps.apple.com/us/app/numbers/id361304891"
    deployment_type       = "Install Automatically/Prompt Users to Install"
    deploy_as_managed_app = true
  }

  scope = {
    targets = {
      mobile_device_group_ids = [
        jamfplatform_device_group.field_ipads.jamf_pro_id,
      ]
    }

    exclusions = {
      department_ids = [jamfplatform_pro_department.it.id]
    }
  }

  app_configuration = {
    preferences = <<-EOT
      <dict>
        <key>ServerURL</key>
        <string>https://example.com</string>
      </dict>
    EOT
  }
}
