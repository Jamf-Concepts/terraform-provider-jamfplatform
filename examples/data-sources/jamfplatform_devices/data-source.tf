# All devices
data "jamfplatform_devices" "all" {
}

# Virtual machines based on model identifier
data "jamfplatform_devices" "virtual_machines" {
  filter {
    selector = "modelIdentifier"
    argument = "VirtualMac*"
  }
}

# iPhones with OS version 26 or higher
data "jamfplatform_devices" "iphones_os_26_plus" {
  filter {
    selector = "model"
    argument = "iPhone*"
  }
  filter {
    selector = "operatingSystemVersion"
    operator = ">="
    argument = "26"
  }
}
