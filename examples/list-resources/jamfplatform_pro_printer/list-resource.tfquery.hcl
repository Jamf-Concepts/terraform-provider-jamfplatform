# List every Jamf Pro printer.
list "jamfplatform_pro_printer" "all" {
  provider = jamfplatform
}

# List printers whose name contains the substring "Lab" (case-insensitive).
list "jamfplatform_pro_printer" "lab_printers" {
  provider = jamfplatform

  config {
    filter = {
      name_substring = "Lab"
    }
  }
}
