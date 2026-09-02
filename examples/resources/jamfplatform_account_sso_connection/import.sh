# Copyright Jamf Software LLC 2026
# SPDX-License-Identifier: MPL-2.0

# Connections are imported by the identifier Jamf assigns them.
terraform import jamfplatform_account_sso_connection.corp con_XXXXXXXXXXXXXXXX

# Two attributes cannot be recovered by an import, because Jamf returns neither.
# After importing, set them to match the connection or the next plan will show a
# change:
#
#   client_secret     — never returned by design
#   enabled_products  — the product names come back, the tenant lists do not
