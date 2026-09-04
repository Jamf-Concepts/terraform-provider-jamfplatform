# A tenant holds at most one UEM Connect integration, so the data source takes no
# arguments.
data "jamfplatform_security_cloud_uem_connect" "current" {}

# Alongside the configuration, it reports what the resource leaves out: whether the
# integration is currently reaching Jamf Pro, which Jamf Pro version is behind it,
# and how the last sync went. Those change on their own, so they belong to a read
# rather than to managed state.
output "uem_connect_health" {
  value = {
    connected        = data.jamfplatform_security_cloud_uem_connect.current.connected
    jamf_pro_version = data.jamfplatform_security_cloud_uem_connect.current.jamf_pro_version
    last_sync_status = try(data.jamfplatform_security_cloud_uem_connect.current.latest_sync.status, "never synced")
  }
}

# Reading the ID is how you find what to import when the integration was set up in
# the Jamf Security Cloud admin UI.
output "uem_connect_id" {
  value = data.jamfplatform_security_cloud_uem_connect.current.id
}
