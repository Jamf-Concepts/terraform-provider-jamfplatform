# Minimal record: serial number and device type only.
resource "jamfplatform_pro_inventory_preload_record" "minimal" {
  serial_number = "C02ZTF001ABC"
  device_type   = "Computer"
}

# Full record with user assignment, purchasing details, and extension attributes.
resource "jamfplatform_pro_inventory_preload_record" "full" {
  serial_number = "C02ZTF002DEF"
  device_type   = "Computer"

  username      = "jappleseed"
  full_name     = "Jamie Appleseed"
  email_address = "jamie.appleseed@example.com"
  phone_number  = "555-0100"
  position      = "Technician"

  department = "IT"
  building   = "HQ"
  room       = "101"

  po_number           = "PO-2026-001"
  po_date             = "2026-01-15"
  warranty_expiration = "2029-01-15"
  lease_expiration    = "2028-01-15"
  purchase_price      = "1999.00"
  purchasing_contact  = "Procurement Team"
  purchasing_account  = "ACCT-100"
  apple_care_id       = "AC-0001"
  life_expectancy     = "4"
  asset_tag           = "ASSET-0001"
  bar_code_1          = "BC-0001"
  vendor              = "Example Reseller"

  extension_attributes = [
    {
      name  = "Cost Center"
      value = "CC-100"
    },
    {
      name  = "Owner Verified"
      value = "Yes"
    },
  ]
}

# Mobile device record.
resource "jamfplatform_pro_inventory_preload_record" "tablet" {
  serial_number = "DMPZTF003GHI"
  device_type   = "Mobile Device"
  username      = "jappleseed"
  building      = "HQ"
}
