# Look up an Inventory Preload record by ID.
data "jamfplatform_pro_inventory_preload_record" "by_id" {
  id = "12"
}

# Look up an Inventory Preload record by device serial number.
data "jamfplatform_pro_inventory_preload_record" "by_serial" {
  serial_number = "C02ZTF001ABC"
}

output "preload_record_username" {
  value = data.jamfplatform_pro_inventory_preload_record.by_serial.username
}
