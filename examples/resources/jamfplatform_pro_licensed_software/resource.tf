# Manage a Jamf Pro licensed software record. software_definitions and licenses
# are ordered lists reconciled by position (the endpoint assigns no per-element
# id). The Jamf Pro API does not support the legacy font / plug-in definition
# buckets.
resource "jamfplatform_pro_licensed_software" "example" {
  name                                    = "Acme Editor"
  publisher                               = "Acme Corp"
  platform                                = "Mac"
  notes                                   = "Managed by Terraform."
  send_email_on_violation                 = true
  remove_titles_from_inventory_reports    = false
  exclude_titles_purchased_from_app_store = false

  software_definitions = [
    {
      name         = "Acme Editor"
      version      = "2.0"
      compare_type = "is"
    },
    {
      name         = "Acme Viewer"
      compare_type = "like"
    },
  ]

  licenses = [
    {
      serial_number_1   = "SER-0001"
      organization_name = "Acme Corp"
      registered_to     = "IT Department"
      license_type      = "Standard"
      license_count     = 25

      purchasing = {
        license_term       = "perpetual"
        po_number          = "PO-12345"
        po_date            = "2026-03-15"
        vendor             = "Acme Reseller"
        license_expires    = "2027-03-15"
        purchase_price     = "1999.00"
        life_expectancy    = 3
        purchasing_account = "Finance"
        purchasing_contact = "Jane Doe"
      }
    },
  ]
}
