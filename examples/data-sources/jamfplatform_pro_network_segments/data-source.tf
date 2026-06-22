data "jamfplatform_pro_network_segments" "all" {}

data "jamfplatform_pro_network_segments" "hq_like" {
  filter = {
    name_substring = "HQ"
  }
}

output "all_network_segments" {
  value = data.jamfplatform_pro_network_segments.all.network_segments
}

output "hq_network_segments" {
  value = data.jamfplatform_pro_network_segments.hq_like.network_segments
}
