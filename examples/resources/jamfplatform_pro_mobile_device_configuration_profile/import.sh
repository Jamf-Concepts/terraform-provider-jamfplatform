# Copyright Jamf Software LLC 2026
# SPDX-License-Identifier: MPL-2.0

# Import an existing mobile device configuration profile by its Jamf Pro ID.
#
# Import records every optional block Jamf Pro reports, including the scope and
# the Self Service settings, so the first plan afterwards can propose removing
# what your configuration does not declare. See the "Importing existing objects"
# guide for what applying that plan does.
terraform import jamfplatform_pro_mobile_device_configuration_profile.example "112"
