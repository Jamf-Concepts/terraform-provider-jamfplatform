action "jamfplatform_pro_enable_lost_mode" "enable" {
  config {
    management_ids = ["00000000-0000-0000-0000-000000000000"]

    lost_mode_message  = "This device is lost. Please return it."
    lost_mode_phone    = "+1-555-0100"
    lost_mode_footnote = "Property of Example Corp"
  }
}
