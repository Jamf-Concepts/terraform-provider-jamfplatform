data "jamfplatform_inventory_mobile_devices" "all" {}

output "devices" {
  value = data.jamfplatform_inventory_mobile_devices.all
}
