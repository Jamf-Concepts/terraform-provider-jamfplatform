# Copyright Jamf Software LLC 2026
# SPDX-License-Identifier: MPL-2.0

# Import a Cloud Identity Provider by its Jamf Pro ID. The provider type is
# discovered from the registry on import, so you do not supply it.
#
# Import records the attribute mappings Jamf Pro reports even where your
# configuration declares none, so the first plan afterwards can propose removing
# them. See the "Importing existing objects" guide for what applying that plan
# does.
#
# The Google keystore file, its password and the rotation trigger cannot be
# restored on import. Jamf Pro keeps no readable copy of any of them, so
# re-declare all three in your configuration.
terraform import jamfplatform_pro_cloud_identity_provider.example "1"
