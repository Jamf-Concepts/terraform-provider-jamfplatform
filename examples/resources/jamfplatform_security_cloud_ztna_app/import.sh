# Copyright Jamf Software LLC 2026
# SPDX-License-Identifier: MPL-2.0

# Access policy applications are imported by application ID.
terraform import jamfplatform_security_cloud_ztna_app.internal_crm 27f04387-0a12-4f70-9256-eeccc67d7304

# Import does not adopt the Security tab. Jamf Security Cloud always holds all three
# requirements, but Terraform manages only the cards the configuration declares, so an
# imported application starts with no `security` block: write one and all three cards
# show as additions even though nothing on the server changes. Read the current
# settings with the jamfplatform_security_cloud_ztna_app data source, then declare the
# cards you want to manage.
