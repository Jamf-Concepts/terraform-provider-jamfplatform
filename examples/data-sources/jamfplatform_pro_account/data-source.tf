data "jamfplatform_pro_account" "by_id" {
  id = "12"
}

data "jamfplatform_pro_account" "by_username" {
  username = "jane.auditor"
}

output "account_by_id" {
  value = data.jamfplatform_pro_account.by_id
}

output "account_by_username" {
  value = data.jamfplatform_pro_account.by_username
}
