# Query for the preload record matching a serial number
list "jamfplatform_pro_inventory_preload_record" "by_serial" {
  provider = jamfplatform

  config {
    filter {
      selector = "serialNumber"
      argument = "C02ZTF001ABC"
    }
  }
}

# Query for all computer preload records assigned to a department
# (deviceType uses 0 for Computer and 1 for Mobile Device in filters)
list "jamfplatform_pro_inventory_preload_record" "it_computers" {
  provider = jamfplatform

  config {
    filter {
      selector = "department"
      argument = "IT"
    }
    filter {
      selector = "deviceType"
      argument = "0"
    }
  }
}
