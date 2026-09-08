# Copyright Jamf Software LLC 2026
# SPDX-License-Identifier: MPL-2.0

# Import an existing licensed software record by its numeric Jamf Pro ID.
#
# Import records every optional block Jamf Pro reports, not only the blocks your
# configuration declares, so the first plan afterwards can propose removing what
# you did not declare. See the "Importing existing objects" guide for what
# applying that plan does.
terraform import jamfplatform_pro_licensed_software.example "65"
