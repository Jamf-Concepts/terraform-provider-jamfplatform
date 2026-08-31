# UEM Connect syncs device inventory and group membership from Jamf Pro into Jamf
# Security Cloud, and can signal device risk back to Jamf Pro.
#
# A tenant holds one integration, so this configuration declares one. Where an
# integration already exists, import it rather than adding a second — a second is
# refused. See import.sh.

# Naming the Jamf Pro tenant is the preferred way to authenticate: Jamf Security
# Cloud creates and manages its own credentials on that tenant, so no secret appears
# in your configuration or your state.
#
# The tenant ID is the one value you cannot read off the UEM Connect screen.
# jamfplatform_pro_tenant_id reads it for the Jamf Pro tenant the provider resolves
# to, which under a platform environment is the Jamf Pro tenant in that environment.
data "jamfplatform_pro_tenant_id" "jamf_pro" {}

resource "jamfplatform_security_cloud_uem_connect" "jamf_pro" {
  uem_vendor = "JAMF_PRO"

  platform_tenant = {
    tenant_id = data.jamfplatform_pro_tenant_id.jamf_pro.tenant_id
  }

  # To supply your own credentials instead, replace the platform_tenant block above
  # with an oauth block and the instance address. The two are mutually exclusive.
  #
  #   uem_server_url = "https://your-instance.jamfcloud.com"
  #   oauth = {
  #     client_id     = var.jamf_pro_client_id
  #     client_secret = var.jamf_pro_client_secret
  #
  #     # The secret is write-only: it is sent on apply and never stored in state,
  #     # so there is nothing to compare against. Increment this to send a rotated
  #     # secret — Jamf Security Cloud cannot update the credentials of an
  #     # integration that already exists, so doing so replaces the integration and
  #     # briefly interrupts syncing.
  #     client_secret_wo_version = 1
  #   }

  sync_refresh_interval_minutes = 720

  # Devices Jamf Pro no longer manages are removed after three consecutive syncs
  # without them. Use keep_deleted_or_retired to retain them instead.
  uem_auto_delete_behavior = "remove_deleted_or_unmanaged"
  unmanaged_sync_threshold = 3

  # Send each device's risk level back to Jamf Pro, so Jamf Pro can act on it.
  device_risk_uem_signaling_enabled = true

  # Omit user_data_field_mapping entirely for Jamf's defaults — the same thing the "Use
  # default data field mapping" checkbox selects. Set it to read a field from
  # somewhere else; here, building an address for devices Jamf Pro has no email for.
  user_data_field_mapping = {
    device_name  = "DEVICE_NAME"
    user_name    = "USER_NAME"
    user_id      = "EXTERNAL_USER_ID"
    phone_number = "PHONE_NUMBER"

    email = {
      source                = "SERIAL_NUMBER"
      suffix                = "devices.example.com"
      only_if_email_missing = true
    }
  }

  # Membership is evaluated top to bottom: a device joins the group of the first
  # entry it matches. Devices matching nothing join the default group.
  #
  # uem_group_id names a Jamf Pro group as computer_ or mobile_ followed by the
  # group's number — which composes straight out of a jamfplatform_device_group,
  # whose device_type is already "computer" or "mobile" and whose jamf_pro_id is that
  # number. Better than a hard-coded "computer_12", which means nothing to a reader
  # and silently stops matching if the group is ever recreated.
  #
  # jamf_pro_id is null when the API integration lacks the Device groups Read
  # permission, or when the group is not in Jamf Pro yet, so an apply immediately
  # after creating the group may need a second run.
  #
  # Jamf Security Cloud verifies neither side of a mapping exists, so a wrong ID is
  # accepted and simply never matches — nothing will tell you it is wrong. That is
  # the argument for referencing a jamfplatform_security_cloud_device_group rather
  # than pasting a UUID: Terraform then guarantees the group exists, and creates it
  # before the mapping that names it.
  group_membership_mapping = {
    enabled = true

    # Omit default_security_cloud_group_id entirely and unmatched devices join the
    # built-in "Default Group", which has no ID and so cannot be named here.
    default_security_cloud_group_id = jamfplatform_security_cloud_device_group.unassigned.id

    mappings = [
      {
        uem_group_id            = "${jamfplatform_device_group.executives.device_type}_${jamfplatform_device_group.executives.jamf_pro_id}"
        security_cloud_group_id = jamfplatform_security_cloud_device_group.executives.id
      },
      {
        uem_group_id            = "${jamfplatform_device_group.field_staff.device_type}_${jamfplatform_device_group.field_staff.jamf_pro_id}"
        security_cloud_group_id = jamfplatform_security_cloud_device_group.field_staff.id
      },
    ]
  }
}

# The Jamf Security Cloud side of each mapping. These hold nothing but a name —
# membership comes from the mapping above.
resource "jamfplatform_security_cloud_device_group" "executives" {
  name = "Executives"
}

resource "jamfplatform_security_cloud_device_group" "field_staff" {
  name = "Field Staff"
}

resource "jamfplatform_security_cloud_device_group" "unassigned" {
  name = "Unassigned Devices"
}

resource "jamfplatform_device_group" "executives" {
  name        = "Executives"
  device_type = "computer"
  group_type  = "static"
  members     = []
}

resource "jamfplatform_device_group" "field_staff" {
  name        = "Field Staff"
  device_type = "mobile"
  group_type  = "static"
  members     = []
}
