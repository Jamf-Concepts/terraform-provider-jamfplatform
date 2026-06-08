# VPP assignment distributing an iOS app and a Mac app to a Jamf user group.
resource "jamfplatform_pro_vpp_assignment" "apps" {
  name                 = "Volume Purchasing — Core Apps"
  vpp_admin_account_id = "3"

  # Apple catalog adam IDs. Omit a content attribute to leave the server's
  # current content untouched; set it to [] to clear that content type.
  ios_app_adam_ids = [6444539476]
  mac_app_adam_ids = [409203825]

  scope = {
    jss_user_group_ids = ["1"]

    exclusions = {
      # Directory-service (LDAP) groups are matched by name.
      directory_service_user_group_names = ["LDAP Admins"]
    }
  }
}

# VPP assignment distributing books to all Jamf users.
#
# Note: un-assigning a book removes it from the assignment, but Apple does not
# return or refund the underlying book license to the account.
resource "jamfplatform_pro_vpp_assignment" "books" {
  name                 = "Volume Purchasing — Reading List"
  vpp_admin_account_id = "3"

  ebook_adam_ids = [1234567890]

  scope = {
    all_jss_users = true
  }
}
