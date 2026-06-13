action "jamfplatform_pro_delete_user" "remove_shared_ipad_user" {
  config {
    serial_number = "DMPXXXXXXXXX"

    user_name      = "student01"
    force_deletion = true
  }
}
