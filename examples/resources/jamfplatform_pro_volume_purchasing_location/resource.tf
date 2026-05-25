# Manages a Jamf Pro Volume Purchasing (VPP) location.
#
# `service_token` is `WriteOnly` — the contents of the `.vpptoken` file
# downloaded from Apple Business Manager / Apple School Manager are sent to
# Jamf Pro on writes but never persisted in Terraform state. The file already
# contains a base64-encoded payload, so use `file()` (not `filebase64()`).
# Bump `service_token_wo_version` to rotate the stored token on the next apply.
#
# Create + token-rotating Update perform: POST → Reclaim → poll until the
# Apple-side content sync completes (`last_sync_time != null`) → final GET.
# On large catalogs the sync can take minutes; override the default 30-minute
# create timeout with the `timeouts` block if your tenant needs longer.
resource "jamfplatform_pro_volume_purchasing_location" "prod" {
  name                     = "vpp-prod"
  service_token            = file("${path.module}/tokens/vpp-prod.vpptoken")
  service_token_wo_version = 1

  automatically_populate_purchased_content  = true
  send_notification_when_no_longer_assigned = true
  auto_register_managed_users               = false

  # site_id is optional. Omit to let Jamf Pro decide; the server emits the
  # sentinel "-1" when unset.
  # site_id = "1"

  timeouts {
    create = "60m"
    update = "60m"
  }
}

output "vpp_prod_id" {
  value = jamfplatform_pro_volume_purchasing_location.prod.id
}

# Use the `content` attribute to look up an App Store / iTunes item's
# available licences before assigning it to a device.
output "vpp_prod_purchased_apps" {
  value = [
    for item in jamfplatform_pro_volume_purchasing_location.prod.content : {
      adam_id   = item.adam_id
      name      = item.name
      available = item.license_count_total - item.license_count_in_use
    }
  ]
}
