# Copyright Jamf Software LLC 2026
# SPDX-License-Identifier: MPL-2.0

# The hostname mappings are a single per-tenant collection, so they are always
# imported as "singleton".
terraform import jamfplatform_security_cloud_dns_hostname_mappings.internal singleton
