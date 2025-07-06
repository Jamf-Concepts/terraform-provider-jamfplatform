# Example: Fetch a Jamf Compliance Benchmark by ID

data "jamfcompliancebenchmarkengine_benchmark" "by_id" {
  id = "12345abcde67890fghij1234"
}

output "benchmark_title" {
  value = data.jamfcompliancebenchmarkengine_benchmark.by_id.title
}

output "benchmark_rules" {
  value = [for r in data.jamfcompliancebenchmarkengine_benchmark.by_id.rules : r.title]
}

# Example: Fetch a Jamf Compliance Benchmark by Title

data "jamfcompliancebenchmarkengine_benchmark" "by_title" {
  title = "Example Benchmark Title"
}

output "benchmark_by_title_id" {
  value = data.jamfcompliancebenchmarkengine_benchmark.by_title.id
}

output "benchmark_by_title_rules" {
  value = [for r in data.jamfcompliancebenchmarkengine_benchmark.by_title.rules : r.title]
}
