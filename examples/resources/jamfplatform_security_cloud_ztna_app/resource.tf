# An access policy application defines the traffic Jamf Security Cloud recognises as
# belonging to one enterprise application, and the conditions under which devices may
# reach it. Each one is a row on the Access policy page.

# Categories and predefined application definitions are both maintained by Jamf, so
# resolve them rather than hard-coding an ID or guessing a category's spelling.
data "jamfplatform_security_cloud_content_categories" "all" {}

data "jamfplatform_security_cloud_ztna_predefined_apps" "all" {}

data "jamfplatform_security_cloud_ztna_shared_gateways" "all" {}

locals {
  # category takes a category's display name, not its internal name.
  cloud_storage = one([
    for category in data.jamfplatform_security_cloud_content_categories.all.content_categories :
    category.display_name if category.display_name == "Cloud & File Storage"
  ])

  slack = one([
    for app in data.jamfplatform_security_cloud_ztna_predefined_apps.all.predefined_apps :
    app.id if app.name == "Slack"
  ])

  uk_shared_pool = one([
    for gateway in data.jamfplatform_security_cloud_ztna_shared_gateways.all.shared_gateways :
    gateway.id if gateway.name == "Shared IP Pool: Europe - UK"
  ])
}

# A custom application: you supply the name and every host name.
resource "jamfplatform_security_cloud_ztna_app" "internal_crm" {
  name     = "Internal CRM"
  category = local.cloud_storage

  # hostnames and direct_ips_and_subnets are the traffic matchers, and both clear by
  # removal: drop an entry here and the application stops matching that traffic on the
  # next apply. A custom application with neither is accepted by Jamf Security Cloud
  # and matches nothing at all, so keep at least one populated.

  # A wildcard covers only subdomains, so list the parent alongside it if it needs
  # to match too. Entries must be mutually exclusive: listing both
  # "*.crm.example.com" and "eu.crm.example.com" is rejected.
  hostnames = [
    "crm.example.com",
    "*.crm.example.com",
  ]

  # Only for applications that cannot be reached by host name at all.
  direct_ips_and_subnets = ["10.20.30.0/24"]

  all_device_groups = false

  device_group_ids = [
    jamfplatform_security_cloud_device_group.engineering.id,
    jamfplatform_security_cloud_device_group.contractors.id,
  ]

  routing = {
    traffic_routing = "Encrypt and route via ZTNA"
    gateway_id      = local.uk_shared_pool
    routing_mode    = "Standard"
  }

  # Contractors reach the same application without the tunnel. A group named here
  # must also appear in device_group_ids, and may appear in only one override.
  routing_overrides = [
    {
      device_group_ids = [jamfplatform_security_cloud_device_group.contractors.id]

      routing = {
        traffic_routing = "Direct device routing"
      }
    },
  ]

  # Each block corresponds to one card on the Security tab. Leave a block out and
  # Jamf Security Cloud keeps its own setting for that requirement.
  security = {
    managed_device = {
      enabled = true
    }

    device_risk = {
      enabled            = true
      deny_at_risk_level = "Medium"
    }

    jamf_trust = {
      enabled                   = true
      device_push_notifications = false
    }
  }
}

# A predefined application: the definition owns the name and contributes its own host names,
# which do not appear in hostnames. Anything listed here is an addition to them, and
# an empty hostnames is normal, because the definition's own names still match.
# Only one application per predefined definition is allowed on a tenant.
resource "jamfplatform_security_cloud_ztna_app" "slack" {
  predefined_app_id = local.slack
  category          = local.cloud_storage
  hostnames         = ["slack-proxy.example.com"]
  all_device_groups = true

  routing = {
    traffic_routing = "Direct device routing"
  }
}

resource "jamfplatform_security_cloud_device_group" "engineering" {
  name = "Engineering"
}

resource "jamfplatform_security_cloud_device_group" "contractors" {
  name = "Contractors"
}
