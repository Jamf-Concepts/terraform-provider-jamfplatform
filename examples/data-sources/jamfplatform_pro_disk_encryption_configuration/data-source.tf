data "jamfplatform_pro_disk_encryption_configuration" "example_by_id" {
  id = "1"
}

data "jamfplatform_pro_disk_encryption_configuration" "example_by_name" {
  name = "Individual recovery key"
}

output "disk_encryption_example_by_id" {
  value = data.jamfplatform_pro_disk_encryption_configuration.example_by_id
}

output "disk_encryption_example_by_name" {
  value = data.jamfplatform_pro_disk_encryption_configuration.example_by_name
}
