data "jamfplatform_device" "example_by_id" {
  id = "12345678-90AB-CDEF-1234-567890ABCDEF"
}

data "jamfplatform_device" "example_by_serial" {
  serial_number = "C02ABCDEFGHJK"
}

output "device_by_id" {
  value = data.jamfplatform_device.example_by_id
}

output "device_by_serial" {
  value = data.jamfplatform_device.example_by_serial
}
