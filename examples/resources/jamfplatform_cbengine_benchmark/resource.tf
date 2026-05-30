# Example: Create a Jamf Compliance Benchmark
#
# Targeting accepts either a set of device group Platform IDs (preferred) via
# target_device_groups, or the deprecated single-value target_device_group.

data "jamfplatform_cbengine_rules" "cis_lvl1" {
  baseline_id = "cis_lvl1"
}

resource "jamfplatform_cbengine_benchmark" "cis_lvl1" {
  title              = "CIS Level 1 Benchmark - All Sources, All Rules"
  description        = "Created by Terraform"
  source_baseline_id = "cis_lvl1"

  sources = [
    for s in data.jamfplatform_cbengine_rules.cis_lvl1.sources : {
      branch   = s.branch
      revision = s.revision
    }
  ]

  rules = [
    for r in data.jamfplatform_cbengine_rules.cis_lvl1.rules : {
      id      = r.id
      enabled = r.enabled
    }
  ]

  # Multiple device groups can be targeted simultaneously.
  target_device_groups = [
    "4a36a1fe-e45a-430d-a966-a4d3ac993577",
    "8b9b5fa0-66ed-43d9-9b9a-2b1f3f7d1e21",
  ]
  enforcement_mode = "MONITOR_AND_ENFORCE"
}

resource "jamfplatform_cbengine_benchmark" "custom_cis_lvl1" {
  title              = "CIS Level 1 Benchmark - All Sources, Custom Rules"
  description        = "Time Server and Critical Update Install"
  source_baseline_id = "cis_lvl1"

  sources = [
    for s in data.jamfplatform_cbengine_rules.cis_lvl1.sources : {
      branch   = s.branch
      revision = s.revision
    }
  ]

  rules = [
    {
      id        = "system_settings_time_server_configure"
      enabled   = true
      odv_value = "ntp.jamf.com"
    },
    {
      id      = "system_settings_critical_update_install_enforce"
      enabled = true
    }
  ]

  # target_device_group remains supported for backwards compatibility but is
  # deprecated. Prefer target_device_groups (set form) for new configurations.
  target_device_group = "4a36a1fe-e45a-430d-a966-a4d3ac993577"
  enforcement_mode    = "MONITOR"
}
