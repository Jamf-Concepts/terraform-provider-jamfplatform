# Copyright Jamf Software LLC 2026
# SPDX-License-Identifier: MPL-2.0

# Domains are imported by domain name, not by ID. The ID Jamf assigns is not
# shown anywhere in Jamf Account, and it changes if a domain is removed and
# claimed again — the name is the stable identifier.
terraform import jamfplatform_account_sso_domain.corp example.com
