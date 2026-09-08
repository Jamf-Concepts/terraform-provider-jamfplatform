# Copyright Jamf Software LLC 2026
# SPDX-License-Identifier: MPL-2.0

# Import an existing patch software title by its Jamf Pro ID.
#
# Import records the package assigned to each version Jamf Pro reports, even
# where your configuration declares none, so the first plan afterwards can
# propose removing the entries you did not declare. See the "Importing existing
# objects" guide for what applying that plan does.
terraform import jamfplatform_pro_patch_software_title.example "6"
