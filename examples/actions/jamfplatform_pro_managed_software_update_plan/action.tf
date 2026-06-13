# Enforce a target OS version on the members of a smart or static group.
# Submitting an update plan is a fire-once directive — it is invoked from a resource's
# lifecycle action_trigger (Terraform >= 1.14).

action "jamfplatform_pro_managed_software_update_plan" "enforce_latest" {
  config {
    # Use the jamf_pro_id exported by jamfplatform_device_group.
    group_id    = "1"
    object_type = "COMPUTER_GROUP" # or MOBILE_DEVICE_GROUP

    update_action = "DOWNLOAD_INSTALL"
    version_type  = "LATEST_ANY"

    # Required only when version_type is SPECIFIC_VERSION or CUSTOM_VERSION:
    # specific_version = "15.1"

    # Optional, by update_action:
    # force_install_local_date_time = "2026-07-01T09:00:00" # DOWNLOAD_INSTALL_SCHEDULE
    # max_deferrals                 = 3                      # DOWNLOAD_INSTALL_ALLOW_DEFERRAL
  }
}
