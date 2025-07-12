# Example: Fetch a Jamf Compliance Benchmark by ID

data "jamfplatform_cbengine_benchmark" "by_id" {
  id = "12345abcde67890fghij1234"
}

output "benchmark_title" {
  value = data.jamfplatform_cbengine_benchmark.by_id.title
}

output "benchmark_rules" {
  value = [for r in data.jamfplatform_cbengine_benchmark.by_id.rules : r.title]
}

# Example: Fetch a Jamf Compliance Benchmark by Title

data "jamfplatform_cbengine_benchmark" "by_title" {
  title = "Example Benchmark Title"
}

output "benchmark_by_title_id" {
  value = data.jamfplatform_cbengine_benchmark.by_title.id
}

output "benchmark_by_title_rules" {
  value = [for r in data.jamfplatform_cbengine_benchmark.by_title.rules : r.title]
}
