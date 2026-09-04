# The Jamf Pro tenant identifier for the scope this provider is configured with.
# Takes no arguments.
data "jamfplatform_pro_tenant_id" "this" {}

# Its reason for existing: naming the Jamf Pro tenant to another Jamf product,
# without copying an identifier between consoles by hand.
resource "jamfplatform_security_cloud_uem_connect" "jamf_pro" {
  uem_vendor = "JAMF_PRO"

  platform_tenant = {
    tenant_id = data.jamfplatform_pro_tenant_id.this.tenant_id
  }
}
