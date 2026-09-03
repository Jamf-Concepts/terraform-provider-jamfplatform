# Manages the Jamf Connect deployment and update settings for a single macOS
# configuration profile (Settings → Jamf apps → Jamf Connect).
#
# The configuration profile must already contain a Jamf Connect payload. It
# then appears automatically under Jamf Connect. This resource adopts it by
# the profile's `id` and controls how Jamf Connect is deployed/updated.
# Destroying the resource does NOT remove Jamf Connect from the profile; it
# only stops Terraform managing the deployment settings.

# A macOS configuration profile carrying a Jamf Connect payload.
resource "jamfplatform_pro_macos_configuration_profile" "connect" {
  general = {
    name     = "Jamf Connect Login"
    payloads = file("${path.module}/jamf-connect-login.mobileconfig")
  }
}

resource "jamfplatform_pro_jamf_connect" "example" {
  # Reference the configuration profile's id (not its uuid).
  profile_id = jamfplatform_pro_macos_configuration_profile.connect.id

  # "Maintenance" in the UI: deploy and keep up to date with patch releases.
  # One of NONE, INITIAL_INSTALLATION_ONLY, PATCH_UPDATES, MINOR_AND_PATCH_UPDATES.
  auto_deployment_type = "PATCH_UPDATES"

  # Required unless auto_deployment_type is NONE. Must be a version offered in
  # the Jamf Pro version picker (Jamf validates it against its release catalog).
  version = "2.45.1"
}
