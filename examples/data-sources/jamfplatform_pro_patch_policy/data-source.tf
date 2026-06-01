# Look up a patch policy by ID. Surfaces the general settings, including the
# server-derived release_date / incremental_update / reboot / minimum_os /
# kill_apps fields. Scope and user interaction are not surfaced — manage the
# policy as a resource for that detail.
data "jamfplatform_pro_patch_policy" "example" {
  id = "12"
}

output "patch_policy_target_version" {
  value = data.jamfplatform_pro_patch_policy.example.target_version
}

output "patch_policy_kill_apps" {
  value = data.jamfplatform_pro_patch_policy.example.kill_apps
}
