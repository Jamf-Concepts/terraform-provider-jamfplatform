data "jamfplatform_pro_automated_device_enrollment" "example_by_id" {
  id = "1"
}

data "jamfplatform_pro_automated_device_enrollment" "example_by_name" {
  name = "ade-prod"
}

output "ade_example_by_id" {
  value = data.jamfplatform_pro_automated_device_enrollment.example_by_id
}

output "ade_example_by_name" {
  value = data.jamfplatform_pro_automated_device_enrollment.example_by_name
}
