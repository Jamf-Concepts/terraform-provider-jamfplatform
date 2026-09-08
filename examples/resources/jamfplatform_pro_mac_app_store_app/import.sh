# Copyright Jamf Software LLC 2026
# SPDX-License-Identifier: MPL-2.0

# Import an existing Mac App Store app by its numeric Jamf Pro ID.
#
# Import records every optional block Jamf Pro reports, not only the blocks your
# configuration declares, so the first plan afterwards can propose removing the
# scope, Self Service and volume purchasing blocks you did not declare. Applying
# that plan changes only the state file: Jamf Pro keeps every value it holds
# there. See the "Importing existing objects" guide.
terraform import jamfplatform_pro_mac_app_store_app.example "84"
