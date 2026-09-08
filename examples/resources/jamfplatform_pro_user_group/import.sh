# Copyright Jamf Software LLC 2026
# SPDX-License-Identifier: MPL-2.0

# Import an existing user group by its numeric Jamf Pro ID.
#
# Import records the group members Jamf Pro reports even when your configuration
# does not declare them, so the first plan afterwards can propose removing them.
# See the "Importing existing objects" guide for what applying that plan does.
terraform import jamfplatform_pro_user_group.example "3"
