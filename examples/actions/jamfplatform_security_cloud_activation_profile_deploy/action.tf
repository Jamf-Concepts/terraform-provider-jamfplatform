# Creates a Jamf Security Cloud activation profile's configuration profile in Jamf
# Pro and scopes it to the Jamf Pro groups you name. One deployment per operating
# system, so cover more than one by invoking the action once for each.

# The activation profile code is issued when the profile is created. A profile
# Terraform manages exposes it as its id, so the deploy can take it directly. For
# a profile created in Jamf Security Cloud instead, read the code off the last
# path segment of the profile's deployment page and pass it in as an input.
# There is no way to look a code up by profile name.
resource "jamfplatform_security_cloud_activation_profile" "field_staff" {
  name      = "Field Staff"
  platforms = ["ios", "mac"]

  capabilities = {
    content_controls = true
    network_security = true
  }
}

resource "jamfplatform_device_group" "tablets" {
  name        = "Managed iPads"
  device_type = "mobile"
  group_type  = "static"
}

resource "jamfplatform_device_group" "laptops" {
  name        = "Managed Macs"
  device_type = "computer"
  group_type  = "static"
}

# jamf_pro_group_ids takes bare Jamf Pro group IDs, so jamf_pro_id goes in
# unadorned. Contrast uem_group_id on the
# jamfplatform_security_cloud_uem_connect resource, which wants the same group
# written as "mobile_20". The two are not interchangeable.
action "jamfplatform_security_cloud_activation_profile_deploy" "supervised_ios" {
  config {
    activation_profile_code = jamfplatform_security_cloud_activation_profile.field_staff.id
    os                      = "ios_supervised"
    jamf_pro_group_ids      = [jamfplatform_device_group.tablets.jamf_pro_id]
  }
}

# macOS scopes to computer groups instead. Naming a mobile device group here would
# be refused, and naming this computer group above would be too.
action "jamfplatform_security_cloud_activation_profile_deploy" "macos" {
  config {
    activation_profile_code = jamfplatform_security_cloud_activation_profile.field_staff.id
    os                      = "macos"
    jamf_pro_group_ids      = [jamfplatform_device_group.laptops.jamf_pro_id]
  }
}

# The deploy needs the UEM Connect integration to exist and be connected to Jamf
# Pro. Nothing in the action's arguments names the integration, so the ordering has
# to come from depends_on.
resource "terraform_data" "deploy" {
  depends_on = [jamfplatform_security_cloud_uem_connect.jamf_pro]

  lifecycle {
    action_trigger {
      events = [after_create]
      actions = [
        action.jamfplatform_security_cloud_activation_profile_deploy.supervised_ios,
        action.jamfplatform_security_cloud_activation_profile_deploy.macos,
      ]
    }
  }
}

# Scope only ever accumulates: re-running with a different set of groups adds them
# rather than replacing what is there. Narrow or clear the scope by editing the
# configuration profile in Jamf Pro.
