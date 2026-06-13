action "jamfplatform_pro_flush_mdm_commands" "clear_stuck" {
  config {
    id_type = "computers"
    id      = "123"
    status  = "Pending+Failed"
  }
}
