data "jamfplatform_pro_cloud_distribution_point" "current" {}

output "cloud_distribution_point_type" {
  value = data.jamfplatform_pro_cloud_distribution_point.current.cdn_type
}

output "cloud_distribution_point_connected" {
  value = data.jamfplatform_pro_cloud_distribution_point.current.has_connection_succeeded
}
