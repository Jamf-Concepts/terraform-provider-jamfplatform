# Minimal mobile device configuration profile: name + payload only.
resource "jamfplatform_pro_mobile_device_configuration_profile" "minimal" {
  general = {
    name     = "Minimal Restrictions Profile"
    payloads = file("${path.module}/restrictions.mobileconfig")
  }
}

# Profile scoped to all mobile devices with a per-device exclusion. The payload
# is a Restrictions profile blocking the App Store.
resource "jamfplatform_pro_mobile_device_configuration_profile" "restrictions_all_devices" {
  general = {
    name                                     = "Restrictions – Block App Store"
    description                              = "Prevents users from installing apps via the App Store."
    level                                    = "Device Level"
    distribution_method                      = "Install Automatically"
    redeploy_on_update                       = "Newly Assigned"
    redeploy_days_before_certificate_expires = 0
    payloads                                 = file("${path.module}/restrictions.mobileconfig")
  }

  scope = {
    targets = {
      all_mobile_devices = true
    }
    exclusions = {
      mobile_device_ids = ["12"]
    }
  }
}

# Self Service profile available to specific device groups. The Self
# Service tab requires `distribution_method = "Make Available in Self Service"`.
resource "jamfplatform_pro_mobile_device_configuration_profile" "self_service" {
  general = {
    name                = "Self Service – VPN Configuration"
    distribution_method = "Make Available in Self Service"
    payloads            = file("${path.module}/vpn.mobileconfig")
  }

  scope = {
    targets = {
      mobile_device_group_ids = ["5", "6"]
    }
  }

  self_service = {
    self_service_description = "Installs the corporate VPN configuration."
    feature_on_main_page     = true
    removal_disallowed       = "Never"
    categories = [
      { id = "44" },
    ]
  }
}
