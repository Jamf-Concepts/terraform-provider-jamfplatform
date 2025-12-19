data "jamfplatform_device" "example_by_id" {
  id = "12345678-90AB-CDEF-1234-567890ABCDEF"
}

output "device_by_id" {
  value = data.jamfplatform_device.example_by_id
}
