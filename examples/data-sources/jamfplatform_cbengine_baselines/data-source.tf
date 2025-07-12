# Example: Fetch all available baselines

data "jamfplatform_cbengine_baselines" "all" {}

output "jamfplatform_cbengine_baselines" {
  value = [for b in data.jamfplatform_cbengine_baselines.all.baselines : b.title]
}
