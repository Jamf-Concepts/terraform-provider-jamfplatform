# Copyright Jamf Software LLC 2026
# SPDX-License-Identifier: MPL-2.0

# Connections are imported by the identifier Jamf assigns them.
terraform import jamfplatform_account_sso_connection.corp con_XXXXXXXXXXXXXXXX

# Three attributes cannot be recovered by an import, because Jamf returns none of
# them. After importing, set them to match the connection or the next plan will
# show a change:
#
#   client_secret         — never returned by design
#   enabled_products      — the product names come back, the tenant lists do not
#   enabled_environments  — never returned at all
#
# A connection built with Microsoft's admin-consent flow in the Jamf Account
# console cannot be imported: it has no client registration of its own, so
# Terraform could never write it back. Read it with the
# jamfplatform_account_sso_connection data source instead.
