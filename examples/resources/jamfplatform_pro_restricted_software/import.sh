# Copyright Jamf Software LLC 2026
# SPDX-License-Identifier: MPL-2.0

# Import an existing restricted software record by its numeric Jamf Pro ID.
#
# Import records every optional block Jamf Pro reports, not only the blocks your
# configuration declares, so the first plan afterwards can propose removing the
# scope you did not declare. Applying that plan changes only the state file:
# Jamf Pro keeps every value it holds there. See the "Importing existing
# objects" guide.
terraform import jamfplatform_pro_restricted_software.example "10"
