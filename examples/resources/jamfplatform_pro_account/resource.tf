# A local administrator account with a preset privilege set.
resource "jamfplatform_pro_account" "auditor" {
  username      = "jane.auditor"
  full_name     = "Jane Auditor"
  email_address = "jane.auditor@example.com"
  access_level  = "Full Access"
  privilege_set = "Auditor"

  # WriteOnly password; bump password_wo_version to rotate.
  password            = var.jane_password
  password_wo_version = 1
}

# A local administrator account with a Custom privilege grid. Privileges are
# validated at plan time against the tenant's Administrator catalog, and
# server-added dependency privileges are reconciled out of state.
resource "jamfplatform_pro_account" "helpdesk" {
  username      = "sam.helpdesk"
  full_name     = "Sam Help Desk"
  email_address = "sam.helpdesk@example.com"
  access_level  = "Full Access"
  privilege_set = "Custom"

  password            = var.sam_password
  password_wo_version = 1

  privileges = {
    jamf_pro_server_objects = [
      "Read Computers",
      "Read Mobile Devices",
    ]
    jamf_pro_server_actions = [
      "Send Computer Remote Lock Command",
    ]
  }
}

# A directory-backed account (signs in via an LDAP / cloud identity provider).
# No password is stored locally.
resource "jamfplatform_pro_account" "directory_admin" {
  username       = "dir.admin@example.com"
  full_name      = "Directory Admin"
  email_address  = "dir.admin@example.com"
  access_level   = "Full Access"
  privilege_set  = "Administrator"
  ldap_server_id = 31
}

variable "jane_password" {
  type      = string
  sensitive = true
}

variable "sam_password" {
  type      = string
  sensitive = true
}
