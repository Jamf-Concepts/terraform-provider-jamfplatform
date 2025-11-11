resource "jamfplatform_blueprints_blueprint" "test_custom_declarations" {
  name        = "Terraform Test Custom Declarations ${var.test_id}"
  description = "Managed by Terraform"

  device_groups = [data.jamfpro_group.test_target_computer_group.group_platform_id]

  custom_declarations {

    declaration {
      channel = "SYSTEM"
      kind    = "CONFIGURATION"
      type    = "com.apple.configuration.softwareupdate.settings"
      payload = jsonencode({
        Beta = {
          RequireProgram = {
            Token       = "<beta-token-here>",
            Description = "AppleSeed for IT"
          },
          ProgramEnrollment = "AlwaysOn"
        }
      })
    }

    declaration {
      channel = "USER"
      kind    = "ASSET"
      type    = "com.apple.asset.credential.userpassword"
      payload = jsonencode({
        Reference = {
          DataURL     = "https://somewhere.com/something.plist",
          ContentType = "application/plist"
        }
      })
    }
  }
}
