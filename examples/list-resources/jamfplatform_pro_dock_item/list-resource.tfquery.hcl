# List every Jamf Pro dock item.
list "jamfplatform_pro_dock_item" "all" {
  provider = jamfplatform
}

# List dock items whose name contains the substring "Calc" (case-insensitive).
list "jamfplatform_pro_dock_item" "calculator_like" {
  provider = jamfplatform

  config {
    filter = {
      name_substring = "Calc"
    }
  }
}
