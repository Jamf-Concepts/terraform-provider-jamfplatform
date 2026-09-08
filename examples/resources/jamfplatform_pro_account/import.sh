# Copyright Jamf Software LLC 2026
# SPDX-License-Identifier: MPL-2.0

# Import an existing Jamf Pro user account by its numeric ID.
#
# Import records every optional block Jamf Pro reports, not only the blocks your
# configuration declares, so the first plan afterwards can propose removing what
# you did not declare. See the "Importing existing objects" guide for what
# applying that plan does.
terraform import jamfplatform_pro_account.example "175"
