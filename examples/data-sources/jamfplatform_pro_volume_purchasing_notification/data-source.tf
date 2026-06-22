data "jamfplatform_pro_volume_purchasing_notification" "by_id" {
  id = "43"
}

data "jamfplatform_pro_volume_purchasing_notification" "by_name" {
  name = "Volume Purchasing — Low Licenses"
}

output "notification_recipients" {
  value = data.jamfplatform_pro_volume_purchasing_notification.by_name.external_recipients
}
