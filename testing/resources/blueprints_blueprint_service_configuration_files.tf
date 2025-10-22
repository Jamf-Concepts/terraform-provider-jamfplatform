resource "jamfplatform_blueprints_blueprint" "test_service_configuration_files" {
  name        = "Terraform Test Service Configuration Files ${var.test_id}"
  description = "Managed by Terraform"

  device_groups = [data.jamfpro_group.test_target_computer_group.group_platform_id]

  service_configuration_files {
    service_config_files {
      service_type = "com.apple.sshd"

      data_asset_reference {
        data_url     = "https://example.com/sshd_config.zip"
        hash_sha_256 = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
      }
    }

    service_config_files {
      service_type = "com.apple.pam"

      data_asset_reference {
        data_url     = "https://example.com/sudoers.zip"
        hash_sha_256 = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
      }
    }
  }
}
