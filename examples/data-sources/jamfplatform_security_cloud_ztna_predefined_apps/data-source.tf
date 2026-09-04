# The Jamf-curated Zero Trust Network Access app templates. Read-only, and the same
# for every entitled tenant.
data "jamfplatform_security_cloud_ztna_predefined_apps" "all" {}

# Resolve a template by name so a configuration need not hard-code its identifier.
locals {
  slack = one([
    for app in data.jamfplatform_security_cloud_ztna_predefined_apps.all.predefined_apps :
    app if app.name == "Slack"
  ])
}

# A template bundles the hostnames an app inherits wholesale, so it is worth
# reviewing them before adopting one.
output "slack_template" {
  value = {
    id        = local.slack.id
    hostnames = local.slack.hostnames
  }
}
