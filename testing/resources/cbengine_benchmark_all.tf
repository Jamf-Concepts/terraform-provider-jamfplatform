data "jamfplatform_cbengine_baselines" "test_all_baselines" {
}

data "jamfplatform_cbengine_rules" "test_all_rules" {
  for_each    = { for baseline in data.jamfplatform_cbengine_baselines.test_all_baselines.baselines : baseline.baseline_id => baseline }
  baseline_id = each.value.baseline_id
}

resource "jamfplatform_cbengine_benchmark" "test_all_benchmarks" {
  for_each = { for baseline in data.jamfplatform_cbengine_baselines.test_all_baselines.baselines : baseline.baseline_id => baseline }

  title              = "Terraform Test ${each.value.title} ${var.test_id}"
  description        = "Managed by Terraform - ${each.value.description}"
  source_baseline_id = each.value.baseline_id

  sources = [
    for s in data.jamfplatform_cbengine_rules.test_all_rules[each.key].sources : {
      branch   = s.branch
      revision = s.revision
    }
  ]

  rules = [
    for r in data.jamfplatform_cbengine_rules.test_all_rules[each.key].rules : {
      id      = r.id
      enabled = r.enabled
    }
  ]

  target_device_group = data.jamfpro_group.test_target_computer_group.group_platform_id
  enforcement_mode    = "MONITOR"
}
