data "jamfplatform_cbengine_rules" "test_all_rules" {
  for_each    = { for baseline in data.jamfplatform_cbengine_baselines.test_all_baselines.baselines : baseline.baseline_id => baseline }
  baseline_id = each.value.baseline_id
}
