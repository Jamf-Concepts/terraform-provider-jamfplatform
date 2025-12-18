data "jamfplatform_cbengine_benchmarks" "all" {
}

output "all_cbengine_benchmarks" {
  value = data.jamfplatform_cbengine_benchmarks.all
}
