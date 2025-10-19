resource "jamfplatform_blueprints_blueprint" "test_service_background_tasks" {
  name        = "Terraform Test Service Background Tasks ${var.test_id}"
  description = "Managed by Terraform"

  device_groups = [data.jamfpro_group.test_target_computer_group.group_platform_id]

  service_background_tasks {
    background_tasks {
      task_type        = "user_defined"
      task_description = "Custom background service task"

      executable_asset_reference {
        data_url     = "https://example.com/service-executable.zip"
        hash_sha_256 = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
      }

      launchd_configurations {
        context = "daemon"

        file_asset_reference {
          data_url     = "https://example.com/launchd-daemon.plist"
          hash_sha_256 = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
          content_type = "application/xml"
        }
      }

      launchd_configurations {
        context = "agent"

        file_asset_reference {
          data_url     = "https://example.com/launchd-agent.plist"
          hash_sha_256 = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
          content_type = "application/xml"
        }
      }
    }
  }
}
