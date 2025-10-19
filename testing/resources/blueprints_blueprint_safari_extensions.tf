resource "jamfplatform_blueprints_blueprint" "test_safari_extensions" {
  name        = "Terraform Test Safari Extensions ${var.test_id}"
  description = "Managed by Terraform"

  device_groups = [data.jamfpro_group.test_target_computer_group.group_platform_id]

  safari_extensions {
    managed_extensions {
      extension_id     = "com.example.adblock"
      state            = "Allowed"
      private_browsing = "AlwaysOff"
      allowed_domains {
        domain = "*.company.com"
      }
      denied_domains {
        domain = "*.social-media.com"
      }
    }
  }
}
