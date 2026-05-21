data "jamfplatform_pro_network_segment" "example_by_id" {
  id = "3"
}

data "jamfplatform_pro_network_segment" "example_by_name" {
  name = "HQ Office"
}

output "network_segment_example_by_id" {
  value = data.jamfplatform_pro_network_segment.example_by_id
}

output "network_segment_example_by_name" {
  value = data.jamfplatform_pro_network_segment.example_by_name
}
